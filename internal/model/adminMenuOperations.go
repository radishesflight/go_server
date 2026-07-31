// Package model adminMenuOperations.go - 菜单可访问的 API 路由(权限的最小单位)
//
// 重要变更:每条"操作"现在是一个具体的 HTTP 接口(method + path)
//
//	不再用 "view/add/edit/delete" 这种抽象 code
//
// 权限存储:
//
//	method + path  → 唯一确定一条权限
//	同一个 path 不同 method 算两条不同权限(如 GET 和 PUT /api/system/adminRoles/:id)
//
// 关系:
//
//	admin_menus  1 ── N  admin_menu_operations
//	admin_roles  N ── N  admin_menu_operations  (通过 admin_role_operations,字段名是 route_id)
//
// 中间件校验:
//  1. 用户登录时把角色关联的 routes 拼成 "METHOD /full/path" 列表
//  2. 请求进来时,直接拿 c.Request.Method + " " + c.FullPath() 查这个 set
//  3. c.FullPath() 自动把 /api/system/adminRoles/123 解析回 /api/system/adminRoles/:id
//
// 加新权限流程:
//  1. 在 router 里加 route
//  2. 在 admin_menu_operations 表里加一行(method + path + menu_id + name)
//  3. 在 admin_role_operations 表给需要的角色关联上
//     不需要改任何业务代码
package model

import (
	"go_server/config"
)

type AdminMenuOperations struct {
	BaseModel
	MenuID uint   `gorm:"index;not null" json:"menu_id"`
	Method string `gorm:"size:10;not null;uniqueIndex:idx_op_method_path" json:"method"` // GET / POST / PUT / DELETE
	Path   string `gorm:"size:255;not null;uniqueIndex:idx_op_method_path" json:"path"`  // /api/system/adminRoles/:id
	Name   string `gorm:"size:100;not null" json:"name"`                                 // 中文名,前端显示用
	Sort   int    `gorm:"default:0" json:"sort"`
}

func (AdminMenuOperations) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_menu_operations"
}
