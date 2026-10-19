package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func AdminUsersRoutes(rg *gin.RouterGroup) {
	// 2026-10-19:个人中心接口 — 单独挂 AuthMiddleware,不走 PermissionMiddleware
	//   所有登录用户(普通角色如"前端")都能改自己的资料
	//   必须放 group 外,因为 group 内已经挂了 PermissionMiddleware
	rg.PUT("/system/adminUsers/me",
		middleware.AuthMiddleware(),
		system.UpdateMyProfile,
	)

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
