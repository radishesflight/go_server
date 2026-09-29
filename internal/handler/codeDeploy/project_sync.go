// Package codeDeploy project_sync.go - 业务项目同步入口
//
// 从 JumpServer Core API 拉节点树,增量 upsert 到 business_projects 表
//
// 接口:
//
//	POST /api/codeDeploy/projects/sync   触发同步,返回 {added, updated, removed, total_jms, total_local}
//
// 业务码翻译:
//
//	client 调用失败    → CodeProjectSyncFail(5008)
//	config 未配置      → CodeProjectSyncNoConfig(5009)
//	响应解析失败      → CodeProjectSyncParse(5010)
package codeDeploy

import (
	"strings"

	"go_server/config"
	"go_server/internal/handler"
	"go_server/pkg/jumpserver"

	"github.com/gin-gonic/gin"
)

// SyncProjects 从 JumpServer 同步业务项目
// POST /api/codeDeploy/projects/sync
func SyncProjects(c *gin.Context) {
	cfg := config.AppConfig.Jumpserver
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Token) == "" {
		handler.Error(c, handler.CodeProjectSyncNoConfig, "JumpServer 未配置(base_url / token 缺失),请在 config.yaml 配 jumpserver 段")
		return
	}

	client := jumpserver.NewClient(cfg.BaseURL, cfg.Token, cfg.OrgID)
	res, err := projectSvc.SyncFromJMS(client)
	if err != nil {
		// 简单分类:响应里带 "解析失败" 走 CodeProjectSyncParse,其他走 Fail
		if strings.Contains(err.Error(), "解析失败") {
			handler.Error(c, handler.CodeProjectSyncParse, "JMS 响应解析失败: "+err.Error())
			return
		}
		handler.Error(c, handler.CodeProjectSyncFail, "同步失败: "+err.Error())
		return
	}
	handler.Success(c, res)
}
