package model

import "go_server/config"

// BusinessAssets 业务项目下的资产(从 JMS 拉取)
//
// 关系:
//
//	business_assets.project_id   → business_projects.id  (N:1)
//	business_assets.jms_id       → JMS /api/v1/assets/assets/ 返回的 id(UUID)
//
// 同步策略(SyncFromJMS):
//   - 按 jms_id 匹配
//   - JMS 有,本地有 → UPDATE name/address/platform(可能 JMS 改名了)
//   - JMS 有,本地没有 → INSERT
//   - JMS 没有,本地有 → 软删(不真删,审计用)
//
// 注意:Description/Comment/Protocol 不会从 JMS 拉,保留人工填写
type BusinessAssets struct {
	BaseModel
	ProjectID uint   `gorm:"index;not null" json:"project_id"`
	JmsID     string `gorm:"uniqueIndex;size:64;not null" json:"jms_id"`
	Name      string `gorm:"size:200;not null" json:"name"`
	Address   string `gorm:"size:128" json:"address"` // IP 或域名
	Platform  string `gorm:"size:64" json:"platform"` // Linux / Windows / ...
	OS        string `gorm:"size:64" json:"os"`       // linux / windows / ...
	Protocol  string `gorm:"size:32" json:"protocol"` // ssh / rdp / ...
	Port      int    `gorm:"default:0" json:"port"`
	Comment   string `gorm:"size:255;default:''" json:"comment"`
	Status    int    `gorm:"default:1" json:"status"` // 1=启用 0=禁用
}

func (BusinessAssets) TableName() string {
	return config.AppConfig.Database.Prefix + "business_assets"
}
