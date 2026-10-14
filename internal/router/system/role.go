// Package system role.go - 角色管理路由
//
// 替代旧的 adminRoles.go
// 路由组: /api/system/adminRoles(注意:路径段跟 menu.code 一致,中间件推断 adminRoles:*)
//
// /options 单独挂 AuthMiddleware,不挂 PermissionMiddleware:
//   用于"管理员管理"页面的"角色"列下拉,任何登录用户都能调(不分角色管理权限)。
package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RoleRoutes 角色管理路由
func RoleRoutes(rg *gin.RouterGroup) {
	// 公开下拉接口:只需登录,不查权限
	rg.GET("/system/adminRoles/options", middleware.AuthMiddleware(), system.GetRoleOptions)

	// 角色管理主路由(需要 PermissionMiddleware 校验)
	roles := rg.Group("/system/adminRoles")
	roles.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		roles.GET("/list", system.GetRolesList)
		roles.GET("/:id", system.GetRole)
		roles.POST("", system.CreateRole)
		roles.PUT("/:id", system.UpdateRole)
		roles.DELETE("/:id", system.DeleteRole)
	}
}
