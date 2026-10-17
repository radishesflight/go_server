// Package service aliyun_image_sync.go - 阿里云镜像同步到本地 DB
//
// 业务背景(2026-10-17):用户希望把镜像信息存到 MySQL,前端列表查询走 DB 而不是打 ECS API
// 同步策略:用户点"重新同步"按钮时,后端:
//  1. 拉阿里云(过滤 system + 命名约定,跨 region 聚合)
//  2. 软删本地 DB 旧数据(保留 deleted_at 用于审计,新增用 INSERT)
//  3. 批量 INSERT 新数据
//
// 为什么不用 upsert:全量替换比逐条 upsert 简单可靠;
//
//	镜像本身可能跨 region,删旧 + 插新保证一致性。
package service

import (
	"errors"
	"sync/atomic"
	"time"

	"go_server/config"
	"go_server/internal/model"
	ecspkg "go_server/pkg/aliyun/ecs"
	"gorm.io/gorm"
)

// 同步状态(进程内单例,前端可查)
type syncStatus struct {
	running   atomic.Bool
	lastAt    atomic.Int64 // unix seconds
	lastCount atomic.Int64
	lastError atomic.Pointer[string]
}

var globalSyncStatus syncStatus

// 业务错误
var (
	ErrAliyunImageNotConfig = errors.New("阿里云 ECS 未配置")
)

// SyncImages 触发一次全量同步(拉阿里云 → 删旧 → 插新)
//
//	这是耗时操作(12 秒左右),前端按钮需显示 loading
//	同步过程中 process 内 running=true,完成后 running=false
func SyncImages() error {
	if !globalSyncStatus.running.CompareAndSwap(false, true) {
		return errors.New("已有同步任务在跑,请稍候")
	}
	defer globalSyncStatus.running.Store(false)

	cfg := config.AppConfig.AliyunECS
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		globalSyncStatus.lastError.Store(&[]string{"阿里云 ECS 未配置"}[0])
		return ErrAliyunImageNotConfig
	}

	// 1. 拉阿里云
	c := ecspkg.NewClient(cfg.AccessKey, cfg.SecretKey, cfg.DefaultRegion, cfg.EndpointSuffix)
	self, shared, err := c.ListAllImagesAllRegions()
	if err != nil {
		msg := "拉阿里云失败: " + err.Error()
		globalSyncStatus.lastError.Store(&msg)
		return err
	}

	// 2. 硬删旧(MySQL 5.7 UNIQUE 不支持 partial index,软删会导致下一轮 INSERT 冲突)
	//   不保留审计历史是因为镜像本身就有阿里云原始记录,这里只是缓存副本
	if delErr := model.DB.Unscoped().Where("1=1").Delete(&model.AliyunImage{}).Error; delErr != nil {
		msg := "删旧失败: " + delErr.Error()
		globalSyncStatus.lastError.Store(&msg)
		return delErr
	}

	// 3. 合并 self + shared,批量插入
	all := make([]model.AliyunImage, 0, len(self)+len(shared))
	now := time.Now()
	for _, img := range self {
		all = append(all, imgToModel(img, model.AliyunImageTypeSelf, now))
	}
	for _, img := range shared {
		all = append(all, imgToModel(img, model.AliyunImageTypeOthers, now))
	}

	// 去重(同一 ImageId 在 self/shared 各只保留一份,以 self 优先)
	// 实际过滤逻辑(2026-10-17):用户希望 self + shared 合并,合并时同一 ImageId 去重
	deduped := make(map[string]model.AliyunImage, len(all))
	for _, m := range all {
		if _, ok := deduped[m.ImageID]; !ok {
			deduped[m.ImageID] = m
		}
	}
	all = make([]model.AliyunImage, 0, len(deduped))
	for _, m := range deduped {
		all = append(all, m)
	}
	if len(all) > 0 {
		// 分批 insert,避免单条 SQL 过长
		const batch = 100
		for i := 0; i < len(all); i += batch {
			end := i + batch
			if end > len(all) {
				end = len(all)
			}
			if insErr := model.DB.Create(all[i:end]).Error; insErr != nil {
				msg := "插入失败: " + insErr.Error()
				globalSyncStatus.lastError.Store(&msg)
				return insErr
			}
		}
	}

	// 4. 更新状态
	globalSyncStatus.lastAt.Store(time.Now().Unix())
	globalSyncStatus.lastCount.Store(int64(len(all)))
	empty := ""
	globalSyncStatus.lastError.Store(&empty)
	return nil
}

// SyncStatusResp 同步状态(给前端轮询/展示用)
type SyncStatusResp struct {
	Running   bool   `json:"running"`
	LastAt    int64  `json:"last_at"` // unix seconds,0 = 从未同步
	LastCount int64  `json:"last_count"`
	LastError string `json:"last_error"` // 空=无错
}

func GetSyncStatus() SyncStatusResp {
	resp := SyncStatusResp{
		Running:   globalSyncStatus.running.Load(),
		LastAt:    globalSyncStatus.lastAt.Load(),
		LastCount: globalSyncStatus.lastCount.Load(),
	}
	if p := globalSyncStatus.lastError.Load(); p != nil {
		resp.LastError = *p
	}
	return resp
}

// imgToModel 把 ECS API 返回的 Image 转 DB model
func imgToModel(img ecspkg.Image, imgType string, snapshotAt time.Time) model.AliyunImage {
	return model.AliyunImage{
		ImageID:         img.ImageID,
		ImageName:       img.ImageName,
		ImageType:       imgType,
		ImageOwnerAlias: img.ImageOwnerAlias,
		RegionID:        img.RegionID,
		OSName:          img.OSName,
		Architecture:    img.Architecture,
		SizeGB:          img.Size,
		Status:          img.Status,
		Description:     img.Description,
		SnapshotAt:      snapshotAt,
	}
}

// ListImagesFromDB 列表查询(从本地 DB 读,毫秒级)
//   - region: 空=全部 region
//   - imageType: 空=全部类型("Self" / "Others")
//   - keyword: 模糊匹配 image_id / image_name
func ListImagesFromDB(region, imageType, keyword string) ([]model.AliyunImage, int64, error) {
	db := model.DB.Model(&model.AliyunImage{}).Where("deleted_at IS NULL")
	if region != "" {
		db = db.Where("region_id = ?", region)
	}
	if imageType != "" {
		db = db.Where("image_type = ?", imageType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("image_id LIKE ? OR image_name LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AliyunImage
	if err := db.Order("snapshot_at DESC, image_id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListImageRegionsFromDB 列出已同步的 region(从 DB 读,代替实时 DescribeRegions)
func ListImageRegionsFromDB() ([]string, error) {
	var regions []string
	err := model.DB.Model(&model.AliyunImage{}).
		Where("deleted_at IS NULL").
		Distinct("region_id").
		Order("region_id").
		Pluck("region_id", &regions).Error
	return regions, err
}

// ensure compile time gorm import is used(避免 import 未用报错)
var _ = gorm.ErrRecordNotFound
