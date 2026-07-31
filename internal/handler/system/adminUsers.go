// Package system adminUsers.go - 管理员(用户)管理 HTTP 入口
//
// 接口列表:
//  GET    /api/system/adminUsers/list  列表分页查询
//  GET    /api/system/adminUsers/:id   单条查询
//  POST   /api/system/adminUsers       新增
//  PUT    /api/system/adminUsers/:id   更新
//  DELETE /api/system/adminUsers/:id   删除(软删除)
//
// 业务码翻译表(与 internal/handler/bizcode.go 一致):
//  service.ErrUserNotFound      → CodeUserNotFound   (1004)
//  service.ErrUserNameDuplicate → CodeUserDuplicate  (1005)
//  service.ErrUserPasswordHash  → CodeUnknown        (9999,内部错误)
//
// 角色下拉数据来源:调 getAdminRolesList 接口(/api/system/adminRoles/list)
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// adminUserSvc 用户管理业务入口
var adminUserSvc = service.NewAdminUserService()

// AdminUsersQuery 列表查询参数(query string)
type AdminUsersQuery struct {
	Page   int `form:"page" json:"page"`     // 页码(从 1 开始,默认 1)
	Size   int `form:"size" json:"size"`     // 每页条数(默认 10)
	Status int `form:"status" json:"status"` // 状态筛选(0=全部,1=启用,2=禁用)
}

// CreateAdminUsersReq 创建请求体
type CreateAdminUsersReq struct {
	Username string `json:"username" binding:"required"` // 用户名(必填,唯一)
	Password string `json:"password" binding:"required"` // 密码(必填,会被 bcrypt 加密)
	Email    string `json:"email"`                      // 邮箱(可选)
	Phone    string `json:"phone"`                      // 手机号(可选)
	Status   int    `json:"status"`                     // 状态(1=启用,0=禁用)
	RoleID   uint   `json:"role_id"`                    // 角色 ID(可选,0=未分配)
}

// UpdateAdminUsersReq 更新请求体
type UpdateAdminUsersReq struct {
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status int    `json:"status"`
	RoleID uint   `json:"role_id"`
}

// GetAdminUsersList 分页查询用户列表
func GetAdminUsersList(c *gin.Context) {
	var query AdminUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	list, total, _ := adminUserSvc.GetList(query.Page, query.Size, query.Status)

	handler.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  query.Page,
		"size":  query.Size,
	})
}

// GetAdminUsers 单条用户
func GetAdminUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	u, err := adminUserSvc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			handler.Error(c, handler.CodeUserNotFound, "用户不存在")
		} else {
			handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		}
		return
	}
	handler.Success(c, u)
}

// CreateAdminUsers 创建用户
func CreateAdminUsers(c *gin.Context) {
	var req CreateAdminUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	u, err := adminUserSvc.Create(req.Username, req.Password, req.Email, req.Phone, req.Status, req.RoleID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNameDuplicate):
			handler.Error(c, handler.CodeUserDuplicate, "用户名已存在")
		case errors.Is(err, service.ErrUserPasswordHash):
			handler.Error(c, handler.CodeUnknown, "密码加密失败")
		default:
			handler.Error(c, handler.CodeUnknown, "创建用户失败")
		}
		return
	}
	handler.Success(c, u)
}

// UpdateAdminUsers 更新用户
func UpdateAdminUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	var req UpdateAdminUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	u, err := adminUserSvc.Update(id, req.Email, req.Phone, req.Status, req.RoleID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			handler.Error(c, handler.CodeUserNotFound, "用户不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "更新用户失败")
		}
		return
	}
	handler.Success(c, u)
}

// DeleteAdminUsers 删除用户
func DeleteAdminUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	if err := adminUserSvc.Delete(id); err != nil {
		handler.Error(c, handler.CodeUnknown, "删除用户失败")
		return
	}
	handler.Success(c, nil)
}
