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
// 权限:走 PermissionMiddleware。
//
//	启动时 SyncRoutes 在 admin_menu_operations 里同步这些路由,
//	并自动给超管角色(role_id=1)关联上;具体见 migrations/2026_08_13_admin_menu_code_deploy.sql
//	和 internal/service/route_sync.go 的白名单。
package codeDeploy

import (
	"go_server/internal/handler/codeDeploy"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// CodeDeployRoutes 注册代码部署路由
func CodeDeployRoutes(rg *gin.RouterGroup) {
	// 全部需登录 + 鉴权
	g := rg.Group("/codeDeploy")
	g.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		// 端字典
		g.GET("/endpoints", codeDeploy.ListEndpoints)

		// 业务项目
		projects := g.Group("/projects")
		{
			projects.GET("/tree", codeDeploy.ListProjectTree)
			projects.GET("", codeDeploy.ListProjects)
			projects.GET("/:id", codeDeploy.GetProject)
			projects.POST("", codeDeploy.CreateProject)
			projects.PUT("/:id", codeDeploy.UpdateProject)
			projects.DELETE("/:id", codeDeploy.DeleteProject)
			projects.POST("/sync", codeDeploy.SyncProjects) // 从 JumpServer 同步
		}

		// 业务项目 + 资产 + 路径 三层嵌套(给"apk/前端包管理"页左侧用)
		// 注意:必须注册在 /projects/:id 之后? 不,这是独立路径,但要在 /projects/:id/...
		// 子组之外(避免被当成 :id 解析)
		g.GET("/projects-with-paths", codeDeploy.ListProjectsWithAssetsAndPaths)

		// 代码包
		pkgs := g.Group("/packages")
		{
			pkgs.GET("", codeDeploy.ListPackages)
			pkgs.GET("/:id", codeDeploy.GetPackage)
			pkgs.GET("/:id/download-url", codeDeploy.GetDownloadURL)
			pkgs.POST("", codeDeploy.UploadPackage)
			pkgs.POST("/:id/pull", codeDeploy.PullPackage)
			pkgs.DELETE("/:id", codeDeploy.DeletePackage)
		}

		// 业务项目下的资产(从 JMS 同步)
		g.GET("/projects/:id/assets", codeDeploy.ListAssetsByProject)
		// 全部启用资产(代码拉取控制台"选择资产"列表)
		// 必须在 /assets/:id/paths 之前注册,否则 :id 会吞掉 assets
		g.GET("/assets", codeDeploy.ListAssets)
		g.POST("/assets/sync", codeDeploy.SyncAssets)

		// 资产路径(每个资产可绑多个 Linux 目录,只描述意图不执行命令)
		// 注意:/assets/:id/paths 要在 /assets/sync 之前注册,否则 :id=sync 冲突
		g.GET("/assets/:id/paths", codeDeploy.ListAssetPaths)
		g.POST("/assets/:id/paths", codeDeploy.AddAssetPath)
		g.DELETE("/assets/paths/:id", codeDeploy.DeleteAssetPath)

		// 一键同步:项目 + 资产 顺序执行(前端 1 个按钮)
		g.POST("/sync", codeDeploy.SyncAll)
	}
}
