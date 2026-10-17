// Package router serverDeploy/serverDeploy.go - 服务器部署模块路由
//
// 注册在 /api/serverDeploy/* 下,需要鉴权(中间件链由 router.SetupRouter 统一加)
//
// 子路由(2026-10-17 改造:从实时拉阿里云 → 查本地 DB):
//
//	GET    /images                          列镜像(支持 region/type/keyword 过滤)
//	POST   /images/sync                     触发同步(拉阿里云 → 删旧 → 插新)
//	GET    /images/sync/status              查询同步状态
//	GET    /regions                         列已同步的 region(从 DB)
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
		// 镜像列表(从本地 DB,毫秒级)
		g.GET("/images", serverDeploy.ListImages)
		// 同步触发 + 状态查询
		g.POST("/images/sync", serverDeploy.SyncImages)
		g.GET("/images/sync/status", serverDeploy.SyncImagesStatus)
		// region 字典(从本地 DB)
		g.GET("/regions", serverDeploy.ListRegions)
	}
}
