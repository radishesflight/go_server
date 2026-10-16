package model

import "go_server/config"

// CodePackages 代码包
//
// 关联关系(3 个外键必填,新流程不允空):
//
//	code_packages.project_id   → business_projects.id    (N:1)
//	code_packages.asset_id     → business_assets.id      (N:1)
//	code_packages.asset_path_id → business_asset_paths.id (N:1,可 0 表示不指定 cwd)
//
// endpoint_id 是早期字典表字段(苹果/安卓/web/admin),新流程已不写,保留兼容老数据(允许 0)。
type CodePackages struct {
	BaseModel
	ProjectID   uint   `gorm:"index;not null" json:"project_id"`                  // 业务项目 id(从所选 asset 反查)
	EndpointID  uint   `gorm:"index;default:0" json:"endpoint_id"`                // 兼容老字段,新流程不写
	AssetID     uint   `gorm:"index;not null" json:"asset_id"`                    // 资产 id(必填)
	AssetPathID uint   `gorm:"index;not null;default:0" json:"asset_path_id"`     // 资产路径 id(0 表示不切目录)
	Name        string `gorm:"size:128;not null" json:"name"`                     // 前端上传时填的原始名
	Version     string `gorm:"size:32;default:''" json:"version"`                 // 后端自动生成,v<unix>
	FullName    string `gorm:"size:255;default:''" json:"full_name"`              // 拼接: name-version.ext
	Ext         string `gorm:"size:16;default:''" json:"ext"`                     // 文件扩展名
	Size        int64  `gorm:"default:0" json:"size"`                             // 字节
	FileURL     string `gorm:"size:512;not null" json:"file_url"`                 // 对象存储访问地址
	FilePath    string `gorm:"size:255;default:''" json:"file_path"`              // 对象存储路径
	UploaderID  uint   `gorm:"index;not null" json:"uploader_id"`                 // 上传人 user_id
	BuildTime   string `gorm:"type:datetime;not null" json:"build_time"`
	Note        string `gorm:"size:500;default:''" json:"note"`
	Status      int    `gorm:"default:1" json:"status"`                           // 1=有效 0=已下架
}

func (CodePackages) TableName() string {
	return config.AppConfig.Database.Prefix + "code_packages"
}
