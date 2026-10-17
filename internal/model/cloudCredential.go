// Package model cloudCredential.go - 云平台凭证(阿里云/腾讯云 AccessKey + SecretKey)
//
// 2026-10-17 新增,菜单挂 serverDeploy/code=credentials 下,提供增删改查接口。
//
// 字段说明:
//   - name        备注名(同一厂商内唯一),如 "xxx公司OA1"
//   - type        aliyun / tencent
//   - access_key  AK(公钥,可展示给用户)
//   - secret_key  SK(私钥,列表用 mask 隐藏,详情页返回完整)
//   - remark      备注
package model

import (
	"time"

	"go_server/config"

	"gorm.io/gorm"
)

type CloudCredential struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:100;not null;uniqueIndex:uk_name_type,priority:1" json:"name"`
	Type      string         `gorm:"size:16;not null;uniqueIndex:uk_name_type,priority:2;index:idx_type" json:"type"` // aliyun / tencent
	AccessKey string         `gorm:"size:128;not null" json:"access_key"`
	SecretKey string         `gorm:"size:255;not null" json:"secret_key"`
	Remark    string         `gorm:"size:500" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名(加 prefix,与同项目其他 model 一致,2026-10-17)
func (CloudCredential) TableName() string {
	return config.AppConfig.Database.Prefix + "cloud_credentials"
}

// 类型常量(避免字面量散落)
const (
	CloudCredentialTypeAliyun  = "aliyun"
	CloudCredentialTypeTencent = "tencent"
)
