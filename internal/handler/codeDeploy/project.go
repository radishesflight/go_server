// Package codeDeploy project.go - 业务项目 HTTP 入口
//
// 接口列表:
//
//	GET    /api/codeDeploy/projects/tree      整棵树(项目 + 端)
//	GET    /api/codeDeploy/projects           列表
//	GET    /api/codeDeploy/projects/:id       单条(带端)
//	POST   /api/codeDeploy/projects           新建
//	PUT    /api/codeDeploy/projects/:id       更新
//	DELETE /api/codeDeploy/projects/:id       删除
//
// 业务码翻译:
//
//	service.ErrProjectNotFound     → CodeProjectNotFound
//	service.ErrProjectCodeExists   → CodeProjectDuplicate
package codeDeploy

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

var projectSvc = service.NewBusinessProjectService()

// CreateProjectReq 新建/更新业务项目请求体
type CreateProjectReq struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      int    `json:"status"`
	EndpointIDs []uint `json:"endpoint_ids"` // 该项目启用的端 ID 列表
}

// ListProjectTree 整棵树
// GET /api/codeDeploy/projects/tree
func ListProjectTree(c *gin.Context) {
	tree, err := projectSvc.ListTree()
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询项目树失败")
		return
	}
	handler.Success(c, gin.H{"list": tree})
}

// ListProjects 列表
// GET /api/codeDeploy/projects
func ListProjects(c *gin.Context) {
	list, err := projectSvc.List()
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询项目列表失败")
		return
	}
	handler.Success(c, gin.H{"list": list})
}

// GetProject 单条
// GET /api/codeDeploy/projects/:id
func GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的项目ID")
		return
	}
	p, err := projectSvc.GetWithEndpoints(uint(id))
	if err != nil {
		if errors.Is(err, service.ErrProjectNotFound) {
			handler.Error(c, handler.CodeProjectNotFound, "项目不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "查询项目失败")
		}
		return
	}
	handler.Success(c, p)
}

// CreateProject 新建
// POST /api/codeDeploy/projects
func CreateProject(c *gin.Context) {
	var req CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	p, err := projectSvc.Create(req.Code, req.Name, req.Description, req.Sort, req.Status, req.EndpointIDs)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectCodeExists):
			handler.Error(c, handler.CodeProjectDuplicate, "项目编码已存在")
		case errors.Is(err, service.ErrProjectCreate):
			handler.Error(c, handler.CodeUnknown, "创建项目失败")
		default:
			handler.Error(c, handler.CodeUnknown, "创建项目失败")
		}
		return
	}
	handler.Success(c, p)
}

// UpdateProject 更新
// PUT /api/codeDeploy/projects/:id
func UpdateProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的项目ID")
		return
	}
	var req CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	p, err := projectSvc.Update(uint(id), req.Code, req.Name, req.Description, req.Sort, req.Status, req.EndpointIDs)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			handler.Error(c, handler.CodeProjectNotFound, "项目不存在")
		case errors.Is(err, service.ErrProjectCodeExists):
			handler.Error(c, handler.CodeProjectDuplicate, "项目编码已存在")
		default:
			handler.Error(c, handler.CodeUnknown, "更新项目失败")
		}
		return
	}
	handler.Success(c, p)
}

// DeleteProject 删除
// DELETE /api/codeDeploy/projects/:id
func DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的项目ID")
		return
	}
	if err := projectSvc.Delete(uint(id)); err != nil {
		if errors.Is(err, service.ErrProjectNotFound) {
			handler.Error(c, handler.CodeProjectNotFound, "项目不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "删除项目失败")
		}
		return
	}
	handler.Success(c, nil)
}
