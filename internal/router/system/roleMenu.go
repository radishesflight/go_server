// Package system roleMenu.go - 角色菜单权限分配路由
//
// 替代旧的 adminRoles.go 里的 /roleMenus/* 路由
// 路由组: /api/system/roleMenu
// 注意:路由组名"roleMenu"对应权限中间件推断"roleMenu:xxx"(菜单 code 是 "roleMenu")
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
		rm.GET("/roleOperations", system.GetRoleOperationCodes)
		rm.POST("/assign", system.AssignMenusAndOperations)
	}
}
