// Package service cleanup_service.go - 部署数据一键清理
//
// 用途:部署日志页「一键清除部署数据」按钮的后端实现。
// 按用户选的时间范围清理两类数据:
//
//  1. 部署日志(go_operation_log)—— created_at 落在区间内的行物理删除
//  2. 代码包(go_code_packages)—— created_at 落在区间内的行物理删除,
//     同时删对象存储(COS / OSS / MinIO)里对应的数据包文件
//
// 时间区间语义:[start, end),与 operation_log 列表筛选口径一致
// (结束日期只给到天时,折算成次日 0 点,含当天)。
//
// 删除顺序:先删对象存储文件,再删 DB 行。
// 文件删失败时保留该行 —— 避免"DB 没了但文件还在"的孤儿,
// 失败明细回传前端,由人工决定是否重试。
package service

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"go_server/internal/model"
	"go_server/pkg/logger"
)

// 清理相关业务错误
var (
	ErrCleanupLogs     = errors.New("清理部署日志失败")
	ErrCleanupPackages = errors.New("清理代码包失败")
)

// CleanupOptions 清理范围
type CleanupOptions struct {
	Start         time.Time // 区间起点(闭)
	End           time.Time // 区间终点(开)
	ClearLogs     bool      // 是否清部署日志
	ClearPackages bool      // 是否清代码包(含对象存储文件)
}

// CleanupFailure 单个对象文件删除失败
type CleanupFailure struct {
	FilePath string `json:"file_path"`
	Error    string `json:"error"`
}

// CleanupResult 清理结果
type CleanupResult struct {
	LogsDeleted     int              `json:"logs_deleted"`     // 删除的部署日志条数
	PackagesDeleted int              `json:"packages_deleted"` // 删除的代码包记录数
	ObjectsDeleted  int              `json:"objects_deleted"`  // 删除的对象存储文件数
	FailedObjects   []CleanupFailure `json:"failed_objects"`   // 删除失败的文件(对应 DB 行已保留)
}

// DeployCleanupService 部署数据清理
type DeployCleanupService struct{}

// NewDeployCleanupService 构造 DeployCleanupService
func NewDeployCleanupService() *DeployCleanupService {
	return &DeployCleanupService{}
}

// Clear 按时间范围清理部署数据
func (s *DeployCleanupService) Clear(opts CleanupOptions) (*CleanupResult, error) {
	result := &CleanupResult{}

	if opts.ClearLogs {
		deleted, err := s.clearOperationLogs(opts.Start, opts.End)
		if err != nil {
			return result, err
		}
		result.LogsDeleted = deleted
	}

	if opts.ClearPackages {
		if err := s.clearPackages(opts.Start, opts.End, result); err != nil {
			return result, err
		}
	}

	return result, nil
}

// clearOperationLogs 删区间内的部署日志
// go_operation_log 没有软删字段,Delete 即物理删除
func (s *DeployCleanupService) clearOperationLogs(start, end time.Time) (int, error) {
	tx := model.DB.Where("created_at >= ? AND created_at < ?", start, end).
		Delete(&model.OperationLog{})
	if tx.Error != nil {
		logger.L.Error("cleanup: delete operation logs failed",
			zap.Error(tx.Error),
		)
		return 0, ErrCleanupLogs
	}
	return int(tx.RowsAffected), nil
}

// clearPackages 删区间内的代码包 + 对象存储里的数据包文件
func (s *DeployCleanupService) clearPackages(start, end time.Time, result *CleanupResult) error {
	var pkgs []model.CodePackages
	// Unscoped:已软删的行也一起清掉,不留残行
	if err := model.DB.Unscoped().
		Where("created_at >= ? AND created_at < ?", start, end).
		Find(&pkgs).Error; err != nil {
		logger.L.Error("cleanup: query packages failed", zap.Error(err))
		return ErrCleanupPackages
	}

	deletedIDs := make([]uint, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.FilePath != "" {
			if err := deletePackageObject(pkg.FilePath); err != nil {
				result.FailedObjects = append(result.FailedObjects, CleanupFailure{
					FilePath: pkg.FilePath,
					Error:    err.Error(),
				})
				logger.L.Warn("cleanup: storage object delete failed, keep db row for retry",
					zap.String("file_path", pkg.FilePath),
					zap.Uint("package_id", pkg.ID),
					zap.Error(err),
				)
				continue // 文件还在,DB 行保留,方便重试
			}
			result.ObjectsDeleted++
		}
		deletedIDs = append(deletedIDs, pkg.ID)
	}

	if len(deletedIDs) == 0 {
		return nil
	}
	// 硬删 DB 行(项目要求不使用软删)
	if err := model.DB.Unscoped().
		Where("id IN ?", deletedIDs).
		Delete(&model.CodePackages{}).Error; err != nil {
		logger.L.Error("cleanup: delete packages failed", zap.Error(err))
		return ErrCleanupPackages
	}
	result.PackagesDeleted = len(deletedIDs)
	return nil
}

// deletePackageObject 删对象存储里的一个数据包文件
func deletePackageObject(filePath string) error {
	uploader := Uploader()
	if uploader == nil {
		return ErrUploadOSSNotConfig
	}
	return uploader.Delete(context.Background(), filePath)
}
