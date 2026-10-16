// Package service business_asset_path_service.go - 业务资产的 Linux 目录路径
//
// 作用:为每个 Linux 资产配置 1..N 个目录路径(后续 SSH 任务可选 cwd)
// 本服务**只描述意图**,不实际执行命令(遵循"JMS 生产环境禁命令"原则)
package service

import (
	"errors"

	"go_server/internal/model"
)

// 业务错误
var (
	ErrAssetPathNotFound  = errors.New("资产路径不存在")
	ErrAssetPathNameEmpty = errors.New("路径名称不能为空")
	ErrAssetPathPathEmpty = errors.New("路径不能为空")
)

// AssetPathService 资产路径服务
type AssetPathService struct{}

// NewAssetPathService 构造
func NewAssetPathService() *AssetPathService { return &AssetPathService{} }

// ListByAsset 查某资产下的所有路径(按 sort asc, id asc)
func (s *AssetPathService) ListByAsset(assetID uint) ([]map[string]interface{}, error) {
	if assetID == 0 {
		return nil, ErrAssetPathNotFound
	}
	var paths []model.BusinessAssetPaths
	if err := model.DB.Where("asset_id = ?", assetID).
		Order("sort asc, id asc").Find(&paths).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(paths))
	for i, p := range paths {
		out[i] = formatAssetPath(p)
	}
	return out, nil
}

func formatAssetPath(p model.BusinessAssetPaths) map[string]interface{} {
	return map[string]interface{}{
		"id":         p.ID,
		"asset_id":   p.AssetID,
		"name":       p.Name,
		"path":       p.Path,
		"sort":       p.Sort,
		"created_at": model.FormatTime(p.CreatedAt),
		"updated_at": model.FormatTime(p.UpdatedAt),
	}
}

// AddPath 给资产加一条路径
func (s *AssetPathService) AddPath(assetID uint, name, path string, sort int) (map[string]interface{}, error) {
	if assetID == 0 {
		return nil, ErrAssetPathNotFound
	}
	if name == "" {
		return nil, ErrAssetPathNameEmpty
	}
	if path == "" {
		return nil, ErrAssetPathPathEmpty
	}
	// 资产必须存在且软删 status=1
	var asset model.BusinessAssets
	if err := model.DB.First(&asset, assetID).Error; err != nil {
		return nil, ErrAssetPathNotFound
	}
	p := model.BusinessAssetPaths{
		AssetID: assetID,
		Name:    name,
		Path:    path,
		Sort:    sort,
	}
	if err := model.DB.Create(&p).Error; err != nil {
		return nil, err
	}
	return formatAssetPath(p), nil
}

// DeletePath 删一条路径(真删,业务表约定不软删)
func (s *AssetPathService) DeletePath(id uint) error {
	if id == 0 {
		return ErrAssetPathNotFound
	}
	return model.DB.Unscoped().Delete(&model.BusinessAssetPaths{}, id).Error
}
