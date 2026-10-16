// Package codeDeploy git_deploy.go - 后端 API 部署入口
//
// 区别于 package.go::PullPackage(走 OSS 包 + 解压):
// 本入口是"git 部署"——目标机器已 git clone 过,部署时直接
//
//	cd $DEPLOY_DIR
//	备份 → 留 5 个
//	git reset --hard origin/{branch}
//	composer install
//	chown -R www.www
//
// 挂在 menu_id=9 "后端API部署" 菜单下
package codeDeploy

import (
	"errors"
	"strconv"
	"time"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// GitDeployReq 后端 API 部署请求体(2026-10-16 v3:path_id 必传,后端校验 type=api)
type GitDeployReq struct {
	PathID uint `json:"path_id" binding:"required"` // 部署路径 id(必须 type=api,且属于本 asset)
}

// GitDeploy 后端 API 部署(走 git reset,不是 OSS 包)
// POST /api/codeDeploy/assets/:id/git-deploy
func GitDeploy(c *gin.Context) {
	start := time.Now()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的资产ID")
		return
	}
	var req GitDeployReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误: "+err.Error())
		return
	}
	if req.PathID == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "请选择项目路径(path_id 必填)")
		return
	}

	// 拿业务资产
	assetSvc := service.NewBusinessAssetService()
	res, err := assetSvc.GitDeploy(uint(id), req.PathID)
	if err != nil {
		// 写 log(失败)
		p := service.LogOperationParams{
			Event: service.EventDeploy, Result: service.ResultFail,
			Method: c.Request.Method, Path: c.FullPath(),
			TargetType: service.TargetAsset, TargetID: uint(id),
			// target_name 用后端服务自拿的 branch(可能为空)
			TargetName: getBranchFromResult(res),
			DurationMS: time.Since(start).Milliseconds(),
			ErrorMsg:   err.Error(),
		}
		if res != nil {
			if v, ok := res["asset_name"].(string); ok {
				p.JMSAssetName = v
			}
			if v, ok := res["jms_id"].(string); ok {
				p.JMSTaskID = v
			}
			// 远端 stderr 优先 summary.failures[asset_name]
			if sm, ok := res["summary"].(map[string]interface{}); ok {
				if fails, ok := sm["failures"].(map[string]interface{}); ok {
					if v, ok := fails[p.JMSAssetName].(string); ok {
						p.JMSOutput = v
					} else {
						for _, v := range fails {
							if s, ok2 := v.(string); ok2 {
								p.JMSOutput = s
								break
							}
						}
					}
				}
			}
			if p.JMSOutput == "" {
				if v, ok := res["jms_output"].(string); ok {
					p.JMSOutput = v
				}
			}
		}
		service.LogOperation(c, p)
		if errors.Is(err, service.ErrAssetNotFound) {
			handler.Error(c, handler.CodeProjectNotFound, "资产不存在")
			return
		}
		// 其他错误(部署失败)用 ErrorWithData 带 stderr 给前端
		handler.ErrorWithData(c, handler.CodePackagePullFail, err.Error(), res)
		return
	}
	// 成功
	pathName := getPathNameFromResult(res)
	p := service.LogOperationParams{
		Event: service.EventDeploy, Result: service.ResultSuccess,
		Method: c.Request.Method, Path: c.FullPath(),
		TargetType: service.TargetAsset, TargetID: uint(id),
		// target_name 用路径名(htapi / 测试 等)而不是 branch 字面量($BRANCH),
		// 让日志列表能按"哪个路径"检索;branch 真实值放 JMSOutput(详情抽屉看)
		TargetName: pathName,
		DurationMS: time.Since(start).Milliseconds(),
	}
	if v, ok := res["asset_name"].(string); ok {
		p.JMSAssetName = v
	}
	if v, ok := res["jms_id"].(string); ok {
		p.JMSTaskID = v
	}
	if v, ok := res["jms_output"].(string); ok {
		p.JMSOutput = v
	}
	service.LogOperation(c, p)
	handler.Success(c, res)
}

// getBranchFromResult 从 res 提取 branch 字段(可能为空)
func getBranchFromResult(res map[string]interface{}) string {
	if res == nil {
		return ""
	}
	if v, ok := res["branch"].(string); ok {
		return v
	}
	return ""
}

// getPathNameFromResult 从 res 提取 path_name 字段(path 节点的名字,如 htapi / 测试)
func getPathNameFromResult(res map[string]interface{}) string {
	if res == nil {
		return ""
	}
	if v, ok := res["path_name"].(string); ok {
		return v
	}
	// fallback: 用 deploy_path
	if v, ok := res["deploy_path"].(string); ok {
		return v
	}
	return ""
}
