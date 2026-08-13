// Package router - HTTP 路由注册
//
// 路由总览(/api 前缀):
//   POST   /api/login                       公开
//   POST   /api/logout                      公开
//   GET    /api/user/info                   需 AuthMiddleware
//   POST   /api/upload/image                需 AuthMiddleware
//   *      /api/system/adminUsers/*         需 Auth + Permission
//   *      /api/system/adminRoles/*         需 Auth + Permission(注意:路径段跟 menu.code 一致)
//   *      /api/system/adminMenus/*         需 Auth + Permission
//   *      /api/system/roleMenu/*           需 Auth + Permission
//   *      /api/system/departments/*        需 Auth + Permission
//   *      /api/codeDeploy/*                需 Auth(+Permission for projects/packages)
//
// 加新路由组的步骤:
//  1. 在 internal/router/<module>/ 加新文件(参考 system/adminUsers.go)
//  2. 在本文件加一行 xxRoutes(api)
package router

import (
	"go_server/internal/handler"
	"go_server/internal/middleware"
	"go_server/internal/router/codeDeploy"
	"go_server/internal/router/system"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化并返回 gin.Engine
// 由 cmd/server/main.go 启动时调一次
//
// 中间件链(顺序重要):
//   gin.Logger    → 访问日志
//   gin.Recovery  → panic 恢复
//   middleware.CORS → 跨域
//   ↓ /api 分组后 ↓
//   (按路由需要)AuthMiddleware      → 鉴权
//   (按路由需要)PermissionMiddleware → 权限校验
//   → handler
func SetupRouter(mode string) *gin.Engine {
	// mode: "debug" / "release" / "test"
	gin.SetMode(mode)

	r := gin.Default()

	// 全局中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 跨域(逻辑在 middleware/cors.go,从 config 读白名单)
	r.Use(middleware.CORS())

	// API 路由组
	api := r.Group("/api")
	{
		// 公开接口
		api.POST("/login", handler.Login)
		api.POST("/logout", handler.Logout)

		// 需登录
		api.GET("/user/info", middleware.AuthMiddleware(), handler.GetCurrentUser)
		api.POST("/upload/image", middleware.AuthMiddleware(), handler.UploadImage)

		// 系统管理(需登录 + 权限)
		system.AdminUsersRoutes(api)
		system.RoleRoutes(api)       // 角色管理
		system.MenuRoutes(api)       // 菜单管理
		system.RoleMenuRoutes(api)   // 角色-菜单-操作 分配
		system.DepartmentRoutes(api) // 部门管理

		// 代码部署
		codeDeploy.CodeDeployRoutes(api)
	}

	return r
}
