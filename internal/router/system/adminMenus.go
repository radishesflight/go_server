package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func AdminMenusRoutes(rg *gin.RouterGroup) {
	adminMenus := rg.Group("/system/adminMenus")
	adminMenus.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		adminMenus.GET("/list", system.GetAdminMenusList)
		// /all 移到 /api/system/adminRoles/allMenus(给"分配菜单"对话框用,推断为 roleMenu:view)
		adminMenus.GET("/options", system.GetAdminMenusOptions)
		adminMenus.GET("/:id", system.GetAdminMenus)
		adminMenus.POST("", system.CreateAdminMenus)
		adminMenus.PUT("/:id", system.UpdateAdminMenus)
		adminMenus.DELETE("/:id", system.DeleteAdminMenus)
	}
}
