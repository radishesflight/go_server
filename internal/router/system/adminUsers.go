package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func AdminUsersRoutes(rg *gin.RouterGroup) {
	adminUsers := rg.Group("/system/adminUsers")
	adminUsers.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		adminUsers.GET("/list", system.GetAdminUsersList)
		adminUsers.GET("/:id", system.GetAdminUsers)
		adminUsers.POST("", system.CreateAdminUsers)
		adminUsers.PUT("/:id", system.UpdateAdminUsers)
		adminUsers.DELETE("/:id", system.DeleteAdminUsers)
	}
}
