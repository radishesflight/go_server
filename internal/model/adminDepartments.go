// Package model adminDepartments.go - 部门表
//
// 用途:给"数据范围权限"的"看部门"用
// 一个用户属于一个部门,部门是树形(parent_id 自引用)
package model

import (
	"go_server/config"
)

type AdminDepartments struct {
	BaseModel
	Name     string `gorm:"size:50;not null" json:"name"`
	ParentID uint   `gorm:"default:0;index" json:"parent_id"`
	Sort     int    `gorm:"default:0" json:"sort"`
	Status   int    `gorm:"default:1" json:"status"`
}

func (AdminDepartments) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_departments"
}
