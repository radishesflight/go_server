package model

import "go_server/config"

// BusinessAssetPaths 业务资产绑定的 Linux 目录路径
//
// 用途:后续 SSH 远程命令执行可选 cwd(给"在 /home/www/web 拉代码"这类任务用)
//
// 关系:
//
//	business_asset_paths.asset_id   → business_assets.id  (N:1)
//
// 一个资产可绑多个路径(比如 web 项目: /home/www/web, /home/www/admin, /data/logs)
//
// 注意:本表**只描述意图**,不实际执行命令 ——
func (BusinessAssetPaths) TableName() string {
	return config.AppConfig.Database.Prefix + "business_asset_paths"
}

type BusinessAssetPaths struct {
	BaseModel
	AssetID uint   `gorm:"index;not null" json:"asset_id"`
	Name    string `gorm:"size:100;not null" json:"name"` // 路径名称(给前端展示用):"前端web目录"
	Path    string `gorm:"size:500;not null" json:"path"` // 实际路径:"/home/www/web"
	Sort    int    `gorm:"default:0" json:"sort"`
}
