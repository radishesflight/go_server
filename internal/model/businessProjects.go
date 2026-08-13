package model

import "go_server/config"

// BusinessProjects 业务项目(树父节点,单层不嵌套)
type BusinessProjects struct {
	BaseModel
	Code        string `gorm:"uniqueIndex;size:64;not null" json:"code"` // 机器名:b2b/dianyao
	Name        string `gorm:"size:64;not null" json:"name"`             // 展示名:b2b电商项目
	Description string `gorm:"size:255;default:''" json:"description"`
	Sort        int    `gorm:"default:0" json:"sort"`
	Status      int    `gorm:"default:1" json:"status"` // 1=启用 0=禁用
}

func (BusinessProjects) TableName() string {
	return config.AppConfig.Database.Prefix + "business_projects"
}
