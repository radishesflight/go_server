package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"go_server/internal/service"
)

// LoginReq 登录请求体(JSON 字段不变,前端无感)
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 登录 / 注销 / 当前用户 业务入口
var authSvc = service.NewAuthService()

// Login 登录
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeParamsInvalid, "用户名和密码不能为空")
		return
	}

	token, user, menus, permissions, err := authSvc.Login(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAuthUserNotFound):
			Error(c, CodeUserNotFound, "用户不存在")
		case errors.Is(err, service.ErrAuthWrongPassword):
			Error(c, CodeUserPassword, "密码错误")
		case errors.Is(err, service.ErrAuthUserNoRole):
			Error(c, CodeUserNoRole, "该用户未分配角色")
		case errors.Is(err, service.ErrAuthRoleNotFound):
			Error(c, CodeRoleNotFound, "角色不存在")
		case errors.Is(err, service.ErrAuthTokenGenFailed):
			Error(c, CodeAuthFail, "令牌生成失败")
		default:
			Error(c, CodeUnknown, "登录失败")
		}
		return
	}

	Success(c, gin.H{
		"token":       token,
		"user":        user,
		"menus":       menus,
		"permissions": permissions,
	})
}

// Logout 注销
func Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		Error(c, CodeTokenMissing, "未携带令牌")
		return
	}

	if err := authSvc.Logout(token); err != nil {
		Error(c, CodeUnknown, "退出登录失败")
		return
	}

	Success(c, nil)
}

// GetCurrentUser 当前用户信息
func GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	roleID, _ := c.Get("role_id")

	uid, _ := userID.(uint)
	rid, _ := roleID.(uint)

	user, menus, permissions, err := authSvc.GetCurrentUser(uid, rid)
	if err != nil {
		Error(c, CodeUnknown, "获取用户信息失败")
		return
	}

	Success(c, gin.H{
		"user":        user,
		"menus":       menus,
		"permissions": permissions,
	})
}
