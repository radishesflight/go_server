package model

import "go_server/config"

// CodePackages 代码包
// 唯一性:同 (project_id, endpoint_id) 下多版本并存(按 build_time desc)
type CodePackages struct {
	BaseModel
	ProjectID  uint   `gorm:"index;not null" json:"project_id"`
	EndpointID uint   `gorm:"index;not null" json:"endpoint_id"`
	Name       string `gorm:"size:128;not null" json:"name"`        // 前端上传时填的原始名
	Version    string `gorm:"size:32;not null" json:"version"`      // 后端自动生成,v2.5.7
	FullName   string `gorm:"size:255;not null" json:"full_name"`   // 拼接: name-version.ext
	Ext        string `gorm:"size:16;not null" json:"ext"`          // 冗余存端 ext
	Size       int64  `gorm:"default:0" json:"size"`                // 字节
	FileURL    string `gorm:"size:512;not null" json:"file_url"`    // OSS 访问地址
	FilePath   string `gorm:"size:255;default:''" json:"file_path"` // OSS 路径
	UploaderID uint   `gorm:"index;not null" json:"uploader_id"`    // 上传人 user_id
	BuildTime  string `gorm:"type:datetime;not null" json:"build_time"`
	Note       string `gorm:"size:500;default:''" json:"note"`
	Status     int    `gorm:"default:1" json:"status"` // 1=有效 0=已下架
}

func (CodePackages) TableName() string {
	return config.AppConfig.Database.Prefix + "code_packages"
}
