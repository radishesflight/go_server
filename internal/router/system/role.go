// Package system role.go - 角色管理路由
//
// 替代旧的 adminRoles.go
// 路由组: /api/system/adminRoles(注意:路径段跟 menu.code 一致,中间件推断 adminRoles:*)
package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RoleRoutes 角色管理路由
func RoleRoutes(rg *gin.RouterGroup) {
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
