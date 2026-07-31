package model

import (
	"go_server/config"
)

type RolePermission struct {
	ID             uint   `gorm:"primarykey" json:"id"`
	RoleID         uint   `gorm:"index;not null" json:"role_id"`
	PermissionCode string `gorm:"size:100;not null" json:"permission_code"`
}

func (RolePermission) TableName() string {
	return config.AppConfig.Database.Prefix + "role_permission"
}
