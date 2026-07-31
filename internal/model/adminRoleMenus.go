// Package model adminRoleMenus.go - 角色-菜单关联
//
// 替代旧的 role_menu_relation 表
// 简洁的 N-N 关联,只存角色-菜单关系
// 操作权限在 admin_role_operations 表单独存
package model

import (
	"go_server/config"
)

type AdminRoleMenus struct {
	ID     uint `gorm:"primarykey" json:"id"`
	RoleID uint `gorm:"index;not null" json:"role_id"`
	MenuID uint `gorm:"index;not null" json:"menu_id"`
}

func (AdminRoleMenus) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_role_menus"
}
