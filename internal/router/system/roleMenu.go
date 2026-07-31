// Package system roleMenu.go - 角色菜单权限分配路由
//
// 路由组: /api/system/roleMenu
// 中间件校验:直接从 token 的 permissions 集合(每项 = "METHOD /path")查 c.Request.Method + " " + c.FullPath()
package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RoleMenuRoutes 角色菜单权限分配路由
func RoleMenuRoutes(rg *gin.RouterGroup) {
	rm := rg.Group("/system/roleMenu")
	rm.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		rm.GET("/allMenus", system.GetRoleMenuAllMenus)
		rm.GET("/roleMenus", system.GetRoleMenuIDs)
		rm.GET("/roleRoutes", system.GetRoleRouteIDs)
		rm.PUT("/assign", system.AssignMenusAndOperations)
	}
}
