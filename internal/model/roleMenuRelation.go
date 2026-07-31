package model

import (
	"go_server/config"
)

type RoleMenuRelation struct {
	ID     uint `gorm:"primarykey" json:"id"`
	RoleID uint `gorm:"index;not null" json:"role_id"`
	MenuID uint `gorm:"index;not null" json:"menu_id"`
}

func (RoleMenuRelation) TableName() string {
	return config.AppConfig.Database.Prefix + "role_menu_relation"
}
