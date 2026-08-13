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
// 路径段说明:
//
//	projects 跟 menu.code 一致,SyncRoutes 会自动反查 menu_id
//	SyncRoutes 启动时把 (method, path) 写入 admin_menu_operations
//
//	提示:路径段必须跟 menu.code 一致(/api/codeDeploy/projects + /api/codeDeploy/endpoints 等)
//	     不要混用下划线或单复数不一致
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
		// 端字典(纯查询,不做权限细控,免得每个新接口都得加 operation 记录)
		g.GET("/endpoints", codeDeploy.ListEndpoints)

		// 业务项目(走权限)
		projects := g.Group("/projects", middleware.PermissionMiddleware())
		{
			projects.GET("/tree", codeDeploy.ListProjectTree)
			projects.GET("", codeDeploy.ListProjects)
			projects.GET("/:id", codeDeploy.GetProject)
			projects.POST("", codeDeploy.CreateProject)
			projects.PUT("/:id", codeDeploy.UpdateProject)
			projects.DELETE("/:id", codeDeploy.DeleteProject)
		}

		// 代码包(走权限)
		pkgs := g.Group("/packages", middleware.PermissionMiddleware())
		{
			pkgs.GET("", codeDeploy.ListPackages)
			pkgs.GET("/:id", codeDeploy.GetPackage)
			pkgs.POST("", codeDeploy.UploadPackage)
			pkgs.POST("/:id/pull", codeDeploy.PullPackage)
		}
	}
}
