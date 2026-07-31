// Package system adminRoles.go - 角色管理 HTTP 入口
//
// 接口列表:
//  GET    /api/system/adminRoles/list          列表分页查询
//  GET    /api/system/adminRoles/:id           单条查询
//  POST   /api/system/adminRoles               新增
//  PUT    /api/system/adminRoles/:id           更新
//  DELETE /api/system/adminRoles/:id           删除
//  GET    /api/system/adminRoles/roleMenus     查某角色的菜单 ID 列表
//  GET    /api/system/adminRoles/roleMenusWithNames  查某角色的菜单名串
//  GET    /api/system/adminRoles/rolePermissions  查某角色的权限码列表
//  POST   /api/system/adminRoles/roleMenus     给某角色分配菜单+权限
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

// adminRoleSvc 角色管理业务入口
var adminRoleSvc = service.NewAdminRoleService()

// AdminRolesQuery 列表查询参数
type AdminRolesQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

// CreateAdminRolesReq 创建请求体
type CreateAdminRolesReq struct {
	Name     string `json:"name" binding:"required"` // 角色名(必填,唯一)
	Describe string `json:"describe"`               // 描述
	Status   int    `json:"status"`                  // 1=启用,0=禁用
}

// UpdateAdminRolesReq 更新请求体
// name 空字符串时不更新(保持原值);describe 无脑覆盖
type UpdateAdminRolesReq struct {
	Name     string `json:"name"`
	Describe string `json:"describe"`
	Status   int    `json:"status"`
}

// GetAdminRolesList 分页查询角色列表
func GetAdminRolesList(c *gin.Context) {
	var query AdminRolesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	list, total, _ := adminRoleSvc.GetList(query.Page, query.Size, query.Status)

	handler.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  query.Page,
		"size":  query.Size,
	})
}

// GetAdminRoles 单条角色
func GetAdminRoles(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	r, err := adminRoleSvc.Get(id)
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

// CreateAdminRoles 创建角色
func CreateAdminRoles(c *gin.Context) {
	var req CreateAdminRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	r, err := adminRoleSvc.Create(req.Name, req.Describe, req.Status)
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

// UpdateAdminRoles 更新角色
func UpdateAdminRoles(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	var req UpdateAdminRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	r, err := adminRoleSvc.Update(id, req.Name, req.Describe, req.Status)
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

// DeleteAdminRoles 删除角色
func DeleteAdminRoles(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	if err := adminRoleSvc.Delete(id); err != nil {
		handler.Error(c, handler.CodeUnknown, "删除角色失败")
		return
	}
	handler.Success(c, nil)
}
