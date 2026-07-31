// Package handler auth.go - 登录 / 注销 / 当前用户 HTTP 入口
//
// 业务处理流程:
//  - Login:        校验参数 → 调 service.Login → 翻译业务码
//  - Logout:       取 token → 调 service.Logout
//  - GetCurrentUser: 从 gin.Context 拿 user_id/role_id(middleware/auth.go 设置的) → service.GetCurrentUser
//
// 业务码翻译表(与 internal/handler/bizcode.go 保持一致):
//  service.ErrAuthUserNotFound  → CodeUserNotFound  (1004)
//  service.ErrAuthWrongPassword → CodeUserPassword  (1006)
//  service.ErrAuthUserNoRole    → CodeUserNoRole    (1007)
//  service.ErrAuthRoleNotFound  → CodeRoleNotFound  (1008)
//  service.ErrAuthTokenGenFailed → CodeAuthFail     (1001)
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"go_server/internal/service"
)

// LoginReq 登录请求体
// 前端 POST /api/login 时传的 JSON
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// authSvc 业务入口(单例,handler 内复用)
var authSvc = service.NewAuthService()

// Login 登录接口
// POST /api/login
// 成功: {code: 0, data: {token, user, menus, permissions}}
// 失败: {code: <业务码>, msg: <错误文案>}
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeParamsInvalid, "用户名和密码不能为空")
		return
	}

	token, user, menus, permissions, err := authSvc.Login(req.Username, req.Password)
	if err != nil {
		// 业务码翻译:service 错误 → handler 业务码
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

// Logout 注销接口
// POST /api/logout
// Header: Authorization: <token>
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

// GetCurrentUser 当前用户信息接口
// GET /api/user/info
// 需要 AuthMiddleware 先跑(否则 gin.Context 里没有 user_id/role_id)
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
