// Package model adminRoleOperations.go - 角色-路由 关联
//
// 替代旧的 (RoleID, MenuID, OperationID) 三元组
// 新设计:每行 = 角色对一条具体路由的访问权(route 自身有 menu_id,不需要再存)
//
// 关系:
//
//	admin_roles N ── N admin_menu_operations  (通过本表,字段 route_id)
package model

import (
	"go_server/config"
)

type AdminRoleOperations struct {
	ID      uint `gorm:"primarykey" json:"id"`
	RoleID  uint `gorm:"index;not null" json:"role_id"`
	RouteID uint `gorm:"index;not null" json:"route_id"` // 指向 admin_menu_operations.id
}

func (AdminRoleOperations) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_role_operations"
}
