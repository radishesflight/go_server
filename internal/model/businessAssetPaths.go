package model

import (
	"time"

	"go_server/config"
)

// BusinessAssetPaths 业务资产绑定的 Linux 目录路径
//
// 用途:后续 SSH 远程命令执行可选 cwd(给"在 /home/www/web 拉代码"这类任务用)
//
// 关系:
//
//	business_asset_paths.asset_id → business_assets.id (N:1)
//
// 一个资产可绑多个路径(比如 web 项目: /home/www/web, /home/www/admin, /data/logs)
//
// 删除策略:本表**真删除**(不软删),沿用项目惯例(同 code_packages)。
//   - 业务层 DeletePath 走 Unscoped().Delete 兜底
//   - 业务层 ListByAsset 不需要带 deleted_at IS NULL 过滤
type BusinessAssetPaths struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime" json:"updated_at"`
	AssetID   uint      `gorm:"index;not null" json:"asset_id"`
	Name      string    `gorm:"size:100;not null" json:"name"` // 路径名称(给前端展示用):"前端web目录"
	Path      string    `gorm:"size:500;not null" json:"path"` // 实际路径:"/home/www/web"
	// 路径类型(2026-10-16 新增):"api" / "web" / "other" / "apk" / "ios"
	//   default 'web' — 新建不指定时按 web,统一前端 web 部署主线
	//   "api"   — 后端 API 代码,走 git reset 部署(Online.vue)
	//   "web"   — 前端 web 包,走 OSS 包上传 + 智能平铺(ApkPackage.vue)
	//   "apk"   — Android 安装包,仅管理不上传(ApkManagement.vue)
	//   "ios"   — iOS 安装包,仅管理不上传(ApkManagement.vue)
	//   "other" — 其它/预留
	//   老的 NULL 数据不补,前端严格 === 过滤自动隐藏
	Type string `gorm:"size:16;index;default:'web'" json:"type"`
	Sort int    `gorm:"default:0" json:"sort"`
}

func (BusinessAssetPaths) TableName() string {
	return config.AppConfig.Database.Prefix + "business_asset_paths"
}
