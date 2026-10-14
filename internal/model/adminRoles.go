package model

import (
	"go_server/config"
)

type AdminRoles struct {
	BaseModel
	Name     string `gorm:"size:50;not null" json:"name"`
	Describe string `gorm:"size:255" json:"describe"`
	Status   int    `gorm:"default:1" json:"status"`
}

func (AdminRoles) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_roles"
}
