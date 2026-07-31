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

// 请求结构(JSON 字段不变,前端无感)
type AdminRolesQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

type CreateAdminRolesReq struct {
	Name     string `json:"name" binding:"required"`
	Describe string `json:"describe"`
	Status   int    `json:"status"`
}

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
