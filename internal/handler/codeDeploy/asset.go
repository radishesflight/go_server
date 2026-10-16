// Package codeDeploy asset.go - 业务资产 HTTP 入口
//
// 接口列表:
//
//	GET  /api/codeDeploy/projects/:id/assets       某项目下的所有资产
//	GET  /api/codeDeploy/assets                    全部启用的资产(代码拉取控制台选资产用)
//	POST /api/codeDeploy/assets/sync               同步(从 JMS 拉,事务 upsert)
//	POST /api/codeDeploy/assets/:id/probe-deps     探测部署依赖(zip/unzip/curl)
//
// 业务码翻译:
//
//	service.ErrAssetSyncFail → CodeProjectSyncFail(5008) 复用
//	其他失败 → CodeUnknown
package codeDeploy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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

// ListAssets 全部启用的资产(代码拉取控制台"选择资产"列表)
// GET /api/codeDeploy/assets
func ListAssets(c *gin.Context) {
	list, err := assetSvc.ListAll()
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询资产列表失败")
		return
	}
	handler.Success(c, gin.H{"list": list})
}

// ProbeAssetDeps 探测指定资产上部署相关命令是否齐全(zip/unzip/curl)
// POST /api/codeDeploy/assets/:id/probe-deps
// 2026-10-16 新增:部署前检查目标机,提前发现缺哪个工具
func ProbeAssetDeps(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的资产ID")
		return
	}
	res, err := assetSvc.ProbeAssetDeps(uint(id))
	if err != nil {
		if errors.Is(err, service.ErrAssetNotFound) {
			handler.Error(c, handler.CodeProjectNotFound, "资产不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "探测失败: "+err.Error())
		}
		return
	}
	handler.Success(c, res)
}

// SyncAssets 从 JMS 同步业务资产(全量,事务)
// POST /api/codeDeploy/assets/sync
func SyncAssets(c *gin.Context) {
	start := time.Now()
	cfg := config.AppConfig.Jumpserver
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Token) == "" {
		handler.Error(c, handler.CodeProjectSyncNoConfig, "JumpServer 未配置")
		return
	}

	client := jumpserver.NewClient(cfg.BaseURL, cfg.Token, cfg.OrgID)
	res, err := assetSvc.SyncFromJMS(client)
	if err != nil {
		service.LogOperation(c, service.LogOperationParams{
			Event: service.EventAssetSync, Result: service.ResultFail,
			Method: c.Request.Method, Path: c.FullPath(),
			TargetType: service.TargetAsset, TargetName: "JMS 业务资产",
			DurationMS: time.Since(start).Milliseconds(),
			ErrorMsg: err.Error(),
		})
		if errors.Is(err, service.ErrAssetSyncFail) {
			handler.Error(c, handler.CodeProjectSyncFail, "同步资产失败: "+err.Error())
			return
		}
		handler.Error(c, handler.CodeProjectSyncFail, "同步失败: "+err.Error())
		return
	}
	tn := "JMS 业务资产"
	if res != nil {
		tn = fmt.Sprintf("JMS 业务资产 +%d ~%d -%d (JMS %d / 本地 %d)",
			res.Added, res.Updated, res.Removed, res.TotalJMS, res.TotalLocal)
	}
	service.LogOperation(c, service.LogOperationParams{
		Event: service.EventAssetSync, Result: service.ResultSuccess,
		Method: c.Request.Method, Path: c.FullPath(),
		TargetType: service.TargetAsset, TargetName: tn,
		DurationMS: time.Since(start).Milliseconds(),
	})
	handler.Success(c, res)
}

// SyncAll 从 JMS 一次性同步"项目+资产"两步(前端 1 个按钮)
//
// POST /api/codeDeploy/sync
// 流程:项目 → 资产(顺序依赖);任一失败整体 fail
func SyncAll(c *gin.Context) {
	start := time.Now()
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
		service.LogOperation(c, service.LogOperationParams{
			Event: service.EventSyncAll, Result: service.ResultFail,
			Method: c.Request.Method, Path: c.FullPath(),
			TargetType: service.TargetProject, TargetName: "JMS 项目+资产",
			DurationMS: time.Since(start).Milliseconds(),
			ErrorMsg: "项目+资产同步都失败: " + projectErr.Error() + " / " + assetErr.Error(),
		})
		handler.Error(c, handler.CodeProjectSyncFail, "项目+资产同步都失败")
		return
	}
	// 至少一个成功,写一条混合 log
	result := service.ResultSuccess
	errMsg := ""
	if projectErr != nil {
		result = service.ResultFail
		errMsg = "项目同步失败: " + projectErr.Error()
	} else if assetErr != nil {
		result = service.ResultFail
		errMsg = "资产同步失败: " + assetErr.Error()
	}
	tn := "JMS 项目+资产"
	if projectRes != nil && assetRes != nil {
		tn = fmt.Sprintf("JMS 项目+资产 项目:+%d ~%d 资产:+%d ~%d", projectRes.Added, projectRes.Updated, assetRes.Added, assetRes.Updated)
	}
	service.LogOperation(c, service.LogOperationParams{
		Event: service.EventSyncAll, Result: result,
		Method: c.Request.Method, Path: c.FullPath(),
		TargetType: service.TargetProject, TargetName: tn,
		DurationMS: time.Since(start).Milliseconds(),
		ErrorMsg: errMsg,
	})
	handler.Success(c, res)
}
