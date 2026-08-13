package model

import "go_server/config"

// BusinessProjectEndpoints 业务项目-端 关联表(项目下启用哪些端,4 选 N)
type BusinessProjectEndpoints struct {
	BaseModel
	ProjectID  uint `gorm:"uniqueIndex:uk_project_endpoint;not null" json:"project_id"`
	EndpointID uint `gorm:"uniqueIndex:uk_project_endpoint;not null" json:"endpoint_id"`
	Sort       int  `gorm:"default:0" json:"sort"`
	Status     int  `gorm:"default:1" json:"status"` // 1=启用 0=禁用
}

func (BusinessProjectEndpoints) TableName() string {
	return config.AppConfig.Database.Prefix + "business_project_endpoints"
}
