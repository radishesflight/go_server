// Package router serverDeploy/serverDeploy.go - 服务器部署模块路由
//
// 注册在 /api/serverDeploy/* 下,需要鉴权(中间件链由 router.SetupRouter 统一加)
//
// 子路由(2026-10-17):
//
//	镜像管理(从 DB,毫秒级):
//	  GET    /images                          列镜像(支持 region/type/keyword 过滤)
//	  POST   /images/sync                     触发同步(拉阿里云 → 删旧 → 插新)
//	  GET    /images/sync/status              查询同步状态
//	  GET    /regions                         列已同步的 region(从 DB)
//
//	云平台凭证管理(2026-10-17 新增,菜单 code=credentials):
//	  GET    /credentials?type=&keyword=      列表(secret_key mask 隐藏)
//	  GET    /credentials/:id                详情(secret_key 明文)
//	  POST   /credentials                    新增
//	  PUT    /credentials/:id                编辑
//	  DELETE /credentials/:id                删除(软删)
package serverDeploy

import (
	"go_server/internal/handler/serverDeploy"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// ServerDeployRoutes 注册服务器部署模块所有路由
//   - rg: 父路由组(一般是 /api,已加过 CORS,AuthMiddleware 由 router.SetupRouter 在外层统一加)
//   - 这里只挂权限校验中间件和具体路由
func ServerDeployRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/serverDeploy")
	g.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		// 镜像管理
		g.GET("/images", serverDeploy.ListImages)
		g.POST("/images/sync", serverDeploy.SyncImages)
		g.GET("/images/sync/status", serverDeploy.SyncImagesStatus)
		g.GET("/regions", serverDeploy.ListRegions)

		// 云平台凭证管理(2026-10-17 新增)
		g.GET("/credentials", serverDeploy.ListCloudCredentials)
		g.GET("/credentials/:id", serverDeploy.GetCloudCredential)
		g.POST("/credentials", serverDeploy.CreateCloudCredential)
		g.PUT("/credentials/:id", serverDeploy.UpdateCloudCredential)
		g.DELETE("/credentials/:id", serverDeploy.DeleteCloudCredential)
	}
}
