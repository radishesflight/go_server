package model

import (
	"go_server/config"
)

type AdminMenus struct {
	BaseModel
	Name      string `gorm:"size:50;not null" json:"name"`
	Code      string `gorm:"size:50;not null;uniqueIndex" json:"code"`
	Path      string `gorm:"size:255" json:"path"`
	Icon      string `gorm:"size:50" json:"icon"`
	ParentID  uint   `gorm:"default:0" json:"parent_id"`
	Sort      int    `gorm:"default:0" json:"sort"`
	Status    int    `gorm:"default:1" json:"status"`
	DataScope int    `gorm:"default:0" json:"data_scope"` // 菜单级数据范围(覆盖角色级,0=全部 1=部门 2=自己)
}

func (AdminMenus) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_menus"
}
