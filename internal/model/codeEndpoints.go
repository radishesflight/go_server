package model

import "go_server/config"

// CodeEndpoints 代码包部署端字典(4 个固定端:苹果/安卓/前台web/后台web)
type CodeEndpoints struct {
	BaseModel
	Code   string `gorm:"uniqueIndex;size:32;not null" json:"code"` // 机器名:ios/android/web/admin
	Name   string `gorm:"size:32;not null" json:"name"`             // 展示名:苹果/安卓/前台web/后台web
	Ext    string `gorm:"size:16;not null" json:"ext"`              // 文件扩展名:apk/zip
	Icon   string `gorm:"size:64;default:''" json:"icon"`           // Element Plus icon 名
	Sort   int    `gorm:"default:0" json:"sort"`
	Status int    `gorm:"default:1" json:"status"` // 1=启用 0=禁用
}

func (CodeEndpoints) TableName() string {
	return config.AppConfig.Database.Prefix + "code_endpoints"
}
