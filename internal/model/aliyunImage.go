// Package model aliyunImage.go - 阿里云 ECS 镜像本地快照
//
// 用途:把阿里云镜像列表同步到本地 MySQL,避免每次列表查询都打阿里云 API(12 秒)
//
// 数据流:
//  1. 前端调 POST /api/serverDeploy/images/sync(触发同步)
//  2. 后端并发拉 33 region 的 self + shared 镜像(过滤 + 去重)
//  3. 删旧 → 插新(全量替换策略,简单)
//  4. 前端列表查询走 DB(支持 region + type 过滤,毫秒级)
//
// 字段约定:
//   - image_id:阿里云 ImageId(全局唯一,跨 region 同一份镜像 ID 一致),做 UNIQUE KEY
//   - image_type:Self / Others(自定义/共享)
//   - image_owner_alias:self / others / system 等
//   - region_id:所属 region
//   - snapshot_at:本行拉取时间(用于过期判断)
package model

import (
	"time"

	"gorm.io/gorm"
)

// AliyunImage 阿里云 ECS 镜像本地快照(2026-10-17 新增)
type AliyunImage struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ImageID         string         `gorm:"size:64;uniqueIndex:uk_image_id;not null" json:"image_id"`
	ImageName       string         `gorm:"size:255;not null" json:"image_name"`
	ImageType       string         `gorm:"size:16;not null;index:idx_region_type,priority:2;default:'Self'" json:"image_type"` // Self / Others
	ImageOwnerAlias string         `gorm:"size:32;not null;default:''" json:"image_owner_alias"`                               // self / system / others / marketplace
	RegionID        string         `gorm:"size:32;not null;index:idx_region_type,priority:1" json:"region_id"`
	OSName          string         `gorm:"size:255;default:''" json:"os_name"`
	Architecture    string         `gorm:"size:16;default:''" json:"architecture"`
	SizeGB          int            `gorm:"default:0" json:"size_gb"`
	Status          string         `gorm:"size:32;default:''" json:"status"`
	Description     string         `gorm:"type:text" json:"description"`
	SnapshotAt      time.Time      `gorm:"not null;index:idx_snapshot" json:"snapshot_at"` // 拉取时间
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名(加 prefix)
func (AliyunImage) TableName() string {
	return "aliyun_images"
}

// 常用查询过滤条件(用 map[string]interface{} 拼,避免拼字符串 SQL)
const (
	AliyunImageTypeSelf   = "Self"
	AliyunImageTypeOthers = "Others"
)
