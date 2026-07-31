package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func AdminRolesRoutes(rg *gin.RouterGroup) {
	adminRoles := rg.Group("/system/adminRoles")
	adminRoles.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		adminRoles.GET("/list", system.GetAdminRolesList)
		adminRoles.GET("/:id", system.GetAdminRoles)
		adminRoles.POST("", system.CreateAdminRoles)
		adminRoles.PUT("/:id", system.UpdateAdminRoles)
		adminRoles.DELETE("/:id", system.DeleteAdminRoles)
		// 角色菜单权限
		// /allMenus 是给"分配菜单"对话框用的(获取所有菜单),推断为 roleMenu:view
		adminRoles.GET("/allMenus", system.GetAllMenus)
		adminRoles.GET("/roleMenus", system.GetMenusByRole)
		adminRoles.GET("/roleMenusWithNames", system.GetMenusByRoleWithNames)
		adminRoles.GET("/rolePermissions", system.GetPermissionsByRole)
		// 分配菜单 = 更新角色菜单/权限,用 PUT 语义,PermissionMiddleware 推断为 adminRoles:edit
		adminRoles.PUT("/roleMenus", system.AssignMenusToRole)
	}
}
