// Package model cloudServer.go - 云服务器本地快照(2026-10-17 新增)
//
// 跟 go_aliyun_images 一个思路:把云厂商 API 的实例数据缓存到本地 DB,
// 前端列表查 DB(毫秒级),按需"同步"按钮刷新。
//
// 设计要点:
//   - credential_id 必填:每条记录挂在某一条 CloudCredential 下,这样多账号(A 主账号,B 子账号)
//     都能各自拉各自的服务器,前端按 credential_id / vendor 过滤展示
//   - vendor 冗余存(aliyun / tencent),方便按厂商筛选不需 join credential 表
//   - 唯一键 (vendor, instance_id):同一厂商内实例唯一(且云厂商内部 instance_id 全局唯一)
//   - 实例创建时间/到期时间可空(老实例或预付费/后付费字段差异)
package model

import (
	"time"

	"go_server/config"

	"gorm.io/gorm"
)

type CloudServer struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CredentialID      uint64         `gorm:"not null;index:idx_credential" json:"credential_id"`
	Vendor            string         `gorm:"size:16;not null;uniqueIndex:uk_vendor_instance,priority:1;index:idx_region" json:"vendor"` // aliyun / tencent
	InstanceID        string         `gorm:"size:64;not null;uniqueIndex:uk_vendor_instance,priority:2" json:"instance_id"`
	Name              string         `gorm:"size:200;not null" json:"name"`
	Region            string         `gorm:"size:64;not null" json:"region"`
	Zone              string         `gorm:"size:64" json:"zone"`
	InstanceType      string         `gorm:"size:64" json:"instance_type"`
	Status            string         `gorm:"size:32;index:idx_status" json:"status"`
	PrivateIP         string         `gorm:"size:64;column:private_ip" json:"private_ip"`
	PublicIP          string         `gorm:"size:64;column:public_ip" json:"public_ip"`
	OSName            string         `gorm:"size:255;column:os_name" json:"os_name"`
	CPU               int            `gorm:"default:0" json:"cpu"`
	MemoryGB          int            `gorm:"default:0;column:memory_gb" json:"memory_gb"`
	InstanceCreatedAt *time.Time     `gorm:"column:instance_created_at" json:"instance_created_at"`
	ExpiredAt         *time.Time     `gorm:"column:expired_at" json:"expired_at"`
	SnapshotAt        time.Time      `gorm:"not null" json:"snapshot_at"` // 本行同步时间
	CreatedAt         time.Time      `json:"created_at"`                  // GORM
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名(加 prefix,2026-10-17)
func (CloudServer) TableName() string {
	return config.AppConfig.Database.Prefix + "cloud_servers"
}

// 厂商常量
const (
	CloudServerVendorAliyun  = "aliyun"
	CloudServerVendorTencent = "tencent"
)
