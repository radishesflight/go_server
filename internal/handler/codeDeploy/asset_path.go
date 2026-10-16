// Package codeDeploy asset_path.go - 资产路径 HTTP 入口
//
// 接口列表:
//
//	GET    /api/codeDeploy/assets/:id/paths        查某资产的所有路径
//	POST   /api/codeDeploy/assets/:id/paths        新增路径
//	DELETE /api/codeDeploy/assets/paths/:id      删除路径
//
// 注意:本模块**只描述意图**,不实际执行 SSH 命令(生产 JMS 禁命令)
package codeDeploy

import (
	"errors"
	"strconv"
	"strings"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

var assetPathSvc = service.NewAssetPathService()

// ListAssetPaths 查某资产的所有路径
// GET /api/codeDeploy/assets/:id/paths?type=api
// type 可选:"api" / "web" / "other",空 = 不过滤(向后兼容)
func ListAssetPaths(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的资产ID")
		return
	}
	pathType := strings.TrimSpace(c.Query("type"))
	list, err := assetPathSvc.ListByAsset(uint(id), pathType)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询资产路径失败")
		return
	}
	handler.Success(c, gin.H{"list": list})
}

// AddPathReq 新增路径请求体(2026-10-16 加 type 字段:api / web / other)
type AddPathReq struct {
	Name string `json:"name" binding:"required"`
	Path string `json:"path" binding:"required"`
	Type string `json:"type"` // "api" / "web" / "other"(默认 other),区分前端/后端/其他
	Sort int    `json:"sort"`
}

// AddAssetPath 新增路径
// POST /api/codeDeploy/assets/:id/paths
func AddAssetPath(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的资产ID")
		return
	}
	var req AddPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误: "+err.Error())
		return
	}
	p, err := assetPathSvc.AddPath(uint(id), req.Name, req.Path, req.Type, req.Sort)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssetPathNameEmpty):
			handler.Error(c, handler.CodeParamsInvalid, "路径名称不能为空")
		case errors.Is(err, service.ErrAssetPathPathEmpty):
			handler.Error(c, handler.CodeParamsInvalid, "路径不能为空")
		case errors.Is(err, service.ErrAssetPathNotFound):
			handler.Error(c, handler.CodeProjectNotFound, "资产不存在")
		default:
			handler.Error(c, handler.CodeUnknown, "新增路径失败")
		}
		return
	}
	handler.Success(c, p)
}

// DeleteAssetPath 删除路径
// DELETE /api/codeDeploy/assets/paths/:id
func DeleteAssetPath(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的路径ID")
		return
	}
	if err := assetPathSvc.DeletePath(uint(id)); err != nil {
		if errors.Is(err, service.ErrAssetPathNotFound) {
			handler.Error(c, handler.CodeProjectNotFound, "路径不存在")
			return
		}
		handler.Error(c, handler.CodeUnknown, "删除路径失败")
		return
	}
	handler.Success(c, nil)
}
