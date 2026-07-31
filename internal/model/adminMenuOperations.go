// Package model adminMenuOperations.go - 菜单支持的操作
//
// 用途:每个菜单可以定义 N 个操作(view/add/edit/delete/import/export/batch_delete/...)
// 加新操作不用改代码,只在表里加一条记录
//
// 权限码生成规则:
//   menu.code + ":" + operation.code  →  "adminUsers:view"、"orders:batch_delete"
//
// 关系:
//   admin_menus  1 ── N  admin_menu_operations
//   admin_roles  N ── N  admin_menu_operations  (通过 admin_role_operations)
package model

import (
	"go_server/config"
)

type AdminMenuOperations struct {
	BaseModel
	MenuID uint   `gorm:"index;not null" json:"menu_id"`
	Code   string `gorm:"size:50;not null" json:"code"` // 操作 code,如 "view"/"add"/"batch_delete"
	Name   string `gorm:"size:50;not null" json:"name"` // 操作名称,前端显示
	Icon   string `gorm:"size:50" json:"icon"`          // 可选,前端展示用
	Sort   int    `gorm:"default:0" json:"sort"`
}

func (AdminMenuOperations) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_menu_operations"
}
