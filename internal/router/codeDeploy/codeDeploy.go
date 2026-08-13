// Package codeDeploy router.go - 代码部署路由注册
//
// 路由总览(/api/codeDeploy 前缀):
//
//	GET    /api/codeDeploy/endpoints                   端字典列表(下拉用)
//	GET    /api/codeDeploy/projects/tree               整棵树(项目+端)
//	GET    /api/codeDeploy/projects                    项目列表
//	GET    /api/codeDeploy/projects/:id                单个项目
//	POST   /api/codeDeploy/projects                    新建项目
//	PUT    /api/codeDeploy/projects/:id                更新项目
//	DELETE /api/codeDeploy/projects/:id                删除项目
//	GET    /api/codeDeploy/packages?project_id=&endpoint_id=  代码包列表
//	GET    /api/codeDeploy/packages/:id                单个代码包
//	POST   /api/codeDeploy/packages                    上传代码包(multipart)
//	POST   /api/codeDeploy/packages/:id/pull           触发部署(目前 mock)
//
// 权限处理(临时方案,流程通了再加回):
//
//	TODO: 走 PermissionMiddleware 前需要
//	  1. 改 route_sync.go:前缀白名单加 "/api/codeDeploy/"
//	  2. admin_menus 表加一条 code='codeDeploy' 菜单(见 migrations/2026_08_13_admin_menu_code_deploy.sql)
//	  3. 重启后端,SyncRoutes 会自动补 11 条 operation + 给超管授权
//	现在先只走 AuthMiddleware,登录就能调。
package codeDeploy

import (
	"go_server/internal/handler/codeDeploy"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// CodeDeployRoutes 注册代码部署路由
func CodeDeployRoutes(rg *gin.RouterGroup) {
	// 全部需登录
	g := rg.Group("/codeDeploy")
	g.Use(middleware.AuthMiddleware())
	{
		// 端字典
		g.GET("/endpoints", codeDeploy.ListEndpoints)

		// 业务项目(权限中间件先注释,流程通了再加回)
		projects := g.Group("/projects" /*, middleware.PermissionMiddleware()*/)
		{
			projects.GET("/tree", codeDeploy.ListProjectTree)
			projects.GET("", codeDeploy.ListProjects)
			projects.GET("/:id", codeDeploy.GetProject)
			projects.POST("", codeDeploy.CreateProject)
			projects.PUT("/:id", codeDeploy.UpdateProject)
			projects.DELETE("/:id", codeDeploy.DeleteProject)
		}

		// 代码包(权限中间件先注释,流程通了再加回)
		pkgs := g.Group("/packages" /*, middleware.PermissionMiddleware()*/)
		{
			pkgs.GET("", codeDeploy.ListPackages)
			pkgs.GET("/:id", codeDeploy.GetPackage)
			pkgs.POST("", codeDeploy.UploadPackage)
			pkgs.POST("/:id/pull", codeDeploy.PullPackage)
		}
	}
}
