package model

import "go_server/config"

// CodePackageDeployTargets 代码包部署目标
//
// 业务:一段代码包可能要部署到多个资产 × 多个路径(比如同一个前端包,推到 5 台机器的 /data/www/web)
// 不再绑定 project/endpoint,只描述"这个包要放到哪些资产/路径上"
//
// 关系:
//
//	code_package_deploy_targets.package_id → code_packages.id
//	code_package_deploy_targets.asset_id  → business_assets.id
//	code_package_deploy_targets.asset_path_id → business_asset_paths.id(可空,表示不指定 cwd)
//
// 唯一性:同 (package_id, asset_id, asset_path_id) 只一条
type CodePackageDeployTargets struct {
	BaseModel
	PackageID   uint `gorm:"index;not null;uniqueIndex:uniq_pkg_asset_path" json:"package_id"`
	AssetID     uint `gorm:"index;not null;uniqueIndex:uniq_pkg_asset_path" json:"asset_id"`
	AssetPathID uint `gorm:"not null;default:0;uniqueIndex:uniq_pkg_asset_path" json:"asset_path_id"`
	Sort        int  `gorm:"default:0" json:"sort"`
	Status      int  `gorm:"default:1" json:"status"` // 1=有效 0=已删
}

func (CodePackageDeployTargets) TableName() string {
	return config.AppConfig.Database.Prefix + "code_package_deploy_targets"
}
