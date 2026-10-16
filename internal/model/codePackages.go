package model

import "go_server/config"

// CodePackages 代码包
//
// 唯一性:不再绑定 (project_id, endpoint_id),改由 code_package_deploy_targets
// 表达"包要部署到哪些资产/路径"。一个包可以被推送到多个目标,project/endpoint 字段
// 保留用于兼容老数据(允许 0)。
type CodePackages struct {
	BaseModel
	ProjectID  uint   `gorm:"index;default:0" json:"project_id"`    // 兼容老字段,新流程可不填
	EndpointID uint   `gorm:"index;default:0" json:"endpoint_id"`   // 兼容老字段,新流程可不填
	Name       string `gorm:"size:128;not null" json:"name"`        // 前端上传时填的原始名
	Version    string `gorm:"size:32;default:''" json:"version"`    // 后端自动生成,v2.5.7
	FullName   string `gorm:"size:255;default:''" json:"full_name"` // 拼接: name-version.ext
	Ext        string `gorm:"size:16;default:''" json:"ext"`        // 文件扩展名
	Size       int64  `gorm:"default:0" json:"size"`                // 字节
	FileURL    string `gorm:"size:512;not null" json:"file_url"`    // 对象存储访问地址
	FilePath   string `gorm:"size:255;default:''" json:"file_path"` // 对象存储路径
	UploaderID uint   `gorm:"index;not null" json:"uploader_id"`    // 上传人 user_id
	BuildTime  string `gorm:"type:datetime;not null" json:"build_time"`
	Note       string `gorm:"size:500;default:''" json:"note"`
	Status     int    `gorm:"default:1" json:"status"` // 1=有效 0=已下架
}

func (CodePackages) TableName() string {
	return config.AppConfig.Database.Prefix + "code_packages"
}
