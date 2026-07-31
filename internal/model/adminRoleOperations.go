// Package model adminRoleOperations.go - 角色-菜单-操作关联
//
// 替代旧的 role_permission 表
// 角色对某个菜单的哪些 operation 有权限
//
// 关系:
//   role_id + menu_id + operation_id  →  唯一确定一条权限
// 权限码生成:menu.code + ":" + operation.code
package model

import (
	"go_server/config"
)

type AdminRoleOperations struct {
	ID          uint `gorm:"primarykey" json:"id"`
	RoleID      uint `gorm:"index;not null" json:"role_id"`
	MenuID      uint `gorm:"index;not null" json:"menu_id"`
	OperationID uint `gorm:"index;not null" json:"operation_id"`
}

func (AdminRoleOperations) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_role_operations"
}
