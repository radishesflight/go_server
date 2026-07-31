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

// 请求结构(JSON 字段不变,前端无感)
type AdminUsersQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

type CreateAdminUsersReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Status   int    `json:"status"`
	RoleID   uint   `json:"role_id"`
}

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
