// Package codeDeploy asset.go - 业务资产 HTTP 入口
//
// 接口列表:
//
//	GET  /api/codeDeploy/projects/:id/assets   某项目下的所有资产
//	POST /api/codeDeploy/assets/sync           同步(从 JMS 拉,事务 upsert)
//
// 业务码翻译:
//
//	service.ErrAssetSyncFail → CodeProjectSyncFail(5008) 复用
//	其他失败 → CodeUnknown
package codeDeploy

import (
	"errors"
	"strconv"
	"strings"

	"go_server/config"
	"go_server/internal/handler"
	"go_server/internal/service"
	"go_server/pkg/jumpserver"

	"github.com/gin-gonic/gin"
)

var assetSvc = service.NewBusinessAssetService()

// ListAssetsByProject 某项目下的所有资产
// GET /api/codeDeploy/projects/:id/assets
func ListAssetsByProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的项目ID")
		return
	}
	list, err := assetSvc.ListByProject(uint(id))
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询资产列表失败")
		return
	}
	handler.Success(c, gin.H{"list": list})
}

// SyncAssets 从 JMS 同步业务资产(全量,事务)
// POST /api/codeDeploy/assets/sync
func SyncAssets(c *gin.Context) {
	cfg := config.AppConfig.Jumpserver
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Token) == "" {
		handler.Error(c, handler.CodeProjectSyncNoConfig, "JumpServer 未配置")
		return
	}

	client := jumpserver.NewClient(cfg.BaseURL, cfg.Token, cfg.OrgID)
	res, err := assetSvc.SyncFromJMS(client)
	if err != nil {
		if errors.Is(err, service.ErrAssetSyncFail) {
			handler.Error(c, handler.CodeProjectSyncFail, "同步资产失败: "+err.Error())
			return
		}
		handler.Error(c, handler.CodeProjectSyncFail, "同步失败: "+err.Error())
		return
	}
	handler.Success(c, res)
}

// SyncAll 从 JMS 一次性同步"项目+资产"两步(前端 1 个按钮)
//
// POST /api/codeDeploy/sync
// 流程:项目 → 资产(顺序依赖);任一失败整体 fail
func SyncAll(c *gin.Context) {
	cfg := config.AppConfig.Jumpserver
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Token) == "" {
		handler.Error(c, handler.CodeProjectSyncNoConfig, "JumpServer 未配置")
		return
	}

	client := jumpserver.NewClient(cfg.BaseURL, cfg.Token, cfg.OrgID)

	// 1. 项目
	projectRes, projectErr := projectSvc.SyncFromJMS(client)

	// 2. 资产(依赖项目已同步)
	assetRes, assetErr := assetSvc.SyncFromJMS(client)

	res := struct {
		Projects   *service.SyncResult      `json:"projects"`
		Assets     *service.AssetSyncResult `json:"assets"`
		ProjectErr string                   `json:"project_err,omitempty"`
		AssetErr   string                   `json:"asset_err,omitempty"`
	}{
		Projects: projectRes,
		Assets:   assetRes,
	}
	if projectErr != nil {
		res.ProjectErr = projectErr.Error()
	}
	if assetErr != nil {
		res.AssetErr = assetErr.Error()
	}
	if projectErr != nil && assetErr != nil {
		handler.Error(c, handler.CodeProjectSyncFail, "项目+资产同步都失败")
		return
	}
	handler.Success(c, res)
}
