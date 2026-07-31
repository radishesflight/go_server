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
		adminRoles.GET("/roleMenus", system.GetMenusByRole)
		adminRoles.GET("/roleMenusWithNames", system.GetMenusByRoleWithNames)
		adminRoles.GET("/rolePermissions", system.GetPermissionsByRole)
		adminRoles.POST("/roleMenus", system.AssignMenusToRole)
	}
}
