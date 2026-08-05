// Package system role.go - 角色管理 HTTP 入口
//
// 替代旧的 adminRoles.go
// 加了 dataScope 字段
//
// 接口列表:
//  GET    /api/system/adminRoles/list    列表分页查询
//  GET    /api/system/adminRoles/:id     单条查询
//  POST   /api/system/adminRoles         新增
//  PUT    /api/system/adminRoles/:id     更新
//  DELETE /api/system/adminRoles/:id     删除
//
// 业务码翻译表:
//  service.ErrRoleNotFound      → CodeRoleNotFound    (1008)
//  service.ErrRoleNameDuplicate → CodeRoleDuplicate   (1009)
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// roleSvc 角色管理业务入口
var roleSvc = service.NewRoleService()

// RolesQuery 列表查询参数
type RolesQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

// CreateRoleReq 创建请求体
type CreateRoleReq struct {
	Name      string `json:"name" binding:"required"`
	Describe  string `json:"describe"`
	Status    int    `json:"status"`
	DataScope int    `json:"data_scope"` // 0=全部 1=部门 2=自己
}

// UpdateRoleReq 更新请求体
type UpdateRoleReq struct {
	Name      string `json:"name"`
	Describe  string `json:"describe"`
	Status    int    `json:"status"`
	DataScope int    `json:"data_scope"`
}

// GetRolesList 分页查询角色列表
func GetRolesList(c *gin.Context) {
	var query RolesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	list, total, _ := roleSvc.GetList(query.Page, query.Size, query.Status)
	handler.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  query.Page,
		"size":  query.Size,
	})
}

// GetRole 单条角色
func GetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	r, err := roleSvc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			handler.Error(c, handler.CodeRoleNotFound, "角色不存在")
		} else {
			handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		}
		return
	}
	handler.Success(c, r)
}

// CreateRole 创建角色
func CreateRole(c *gin.Context) {
	var req CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	r, err := roleSvc.Create(req.Name, req.Describe, req.Status, req.DataScope)
	if err != nil {
		if errors.Is(err, service.ErrRoleNameDuplicate) {
			handler.Error(c, handler.CodeRoleDuplicate, "角色名称已存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "创建角色失败")
		}
		return
	}
	handler.Success(c, r)
}

// UpdateRole 更新角色
func UpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	var req UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	r, err := roleSvc.Update(id, req.Name, req.Describe, req.Status, req.DataScope)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			handler.Error(c, handler.CodeRoleNotFound, "角色不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "更新角色失败")
		}
		return
	}
	handler.Success(c, r)
}

// DeleteRole 删除角色
func DeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	if err := roleSvc.Delete(id); err != nil {
		handler.Error(c, handler.CodeUnknown, "删除角色失败")
		return
	}
	handler.Success(c, nil)
}
