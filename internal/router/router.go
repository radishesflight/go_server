package router

import (
	"go_server/internal/handler"
	"go_server/internal/middleware"
	"go_server/internal/router/system"

	"github.com/gin-gonic/gin"
)

func SetupRouter(mode string) *gin.Engine {
	gin.SetMode(mode)

	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 跨域(逻辑已抽到 middleware/cors.go)
	r.Use(middleware.CORS())

	// API 路由组
	api := r.Group("/api")
	{
		api.POST("/login", handler.Login)
		api.POST("/logout", handler.Logout)
		api.GET("/user/info", middleware.AuthMiddleware(), handler.GetCurrentUser)
		api.POST("/upload/image", middleware.AuthMiddleware(), handler.UploadImage)
		system.AdminUsersRoutes(api)
		system.AdminRolesRoutes(api)
		system.AdminMenusRoutes(api)
	}

	return r
}
