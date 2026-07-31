// Package service auth_service.go - 认证业务
//
// 三个核心方法:
//  1. Login:校验用户名密码 → 查角色 → 查菜单 → 查权限 → 生成 token
//  2. Logout:删 Redis 里的 token 记录
//  3. GetCurrentUser:从 token 重新查用户/菜单/权限(用于刷新)
//
// 业务错误(handler 用 errors.Is 翻译):
//  ErrAuthUserNotFound    → "用户不存在"     CodeUserNotFound
//  ErrAuthWrongPassword   → "密码错误"       CodeUserPassword
//  ErrAuthUserNoRole      → "未分配角色"     CodeUserNoRole
//  ErrAuthRoleNotFound    → "角色不存在"     CodeRoleNotFound
//  ErrAuthTokenGenFailed  → "令牌生成失败"   CodeAuthFail
package service

import (
	"errors"

	"github.com/gin-gonic/gin"

	"go_server/internal/model"
	"go_server/pkg/cache"
)

// 业务错误(与原 handler 中的错误文案一一对应,handler 翻译)
var (
	ErrAuthUserNotFound   = errors.New("用户不存在")
	ErrAuthWrongPassword  = errors.New("密码错误")
	ErrAuthUserNoRole     = errors.New("该用户未分配角色")
	ErrAuthRoleNotFound   = errors.New("角色不存在")
	ErrAuthTokenGenFailed = errors.New("令牌生成失败")
)

// AuthService 登录 / 注销 / 当前用户 业务
type AuthService struct{}

// NewAuthService 构造 AuthService
func NewAuthService() *AuthService { return &AuthService{} }

// Login 处理登录
// 返回:token, user(map), menus(树), permissions
func (s *AuthService) Login(username, password string) (string, gin.H, []*MenuTreeNode, []string, error) {
	var user model.AdminUsers
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", nil, nil, nil, ErrAuthUserNotFound
	}

	if !user.CheckPassword(password) {
		return "", nil, nil, nil, ErrAuthWrongPassword
	}

	if user.RoleID == 0 {
		return "", nil, nil, nil, ErrAuthUserNoRole
	}

	var role model.AdminRoles
	if err := model.DB.Where("id = ?", user.RoleID).First(&role).Error; err != nil {
		return "", nil, nil, nil, ErrAuthRoleNotFound
	}

	var menuIDs []uint
	model.DB.Model(&model.RoleMenuRelation{}).Where("role_id = ?", user.RoleID).Pluck("menu_id", &menuIDs)

	var menus []model.AdminMenus
	model.DB.Where("id IN ? AND status = ?", menuIDs, 1).Order("sort DESC").Find(&menus)

	menuList := BuildMenuTree(menus)

	flatMenus := make([]cache.Menu, 0)
	for _, m := range menus {
		flatMenus = append(flatMenus, cache.Menu{
			ID:   m.ID,
			Name: m.Name,
			Path: m.Path,
			Icon: m.Icon,
		})
	}

	permissions := make([]string, 0)
	var rolePermissions []model.RolePermission
	model.DB.Where("role_id = ?", user.RoleID).Find(&rolePermissions)
	for _, p := range rolePermissions {
		permissions = append(permissions, p.PermissionCode)
	}

	token, err := cache.GenerateToken(user.ID, user.Username, user.RoleID, flatMenus, permissions)
	if err != nil {
		return "", nil, nil, nil, ErrAuthTokenGenFailed
	}

	return token, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"phone":    user.Phone,
		"status":   user.Status,
		"role_id":  user.RoleID,
		"role":     role.Name,
	}, menuList, permissions, nil
}

// GetCurrentUser 取当前用户信息(根据 token 解出的 userID / roleID 重新查 DB)
// 返回:user(map), menus(树), permissions
func (s *AuthService) GetCurrentUser(userID, roleID uint) (gin.H, []*MenuTreeNode, []string, error) {
	var user model.AdminUsers
	model.DB.Where("id = ?", userID).First(&user)

	var role model.AdminRoles
	model.DB.Where("id = ?", roleID).First(&role)

	// 从数据库重新查询菜单
	var menuIDs []uint
	model.DB.Model(&model.RoleMenuRelation{}).Where("role_id = ?", roleID).Pluck("menu_id", &menuIDs)

	var menus []model.AdminMenus
	model.DB.Where("id IN ? AND status = ?", menuIDs, 1).Order("sort DESC").Find(&menus)

	menuList := BuildMenuTree(menus)

	// 从数据库重新查询权限
	permissions := make([]string, 0)
	var rolePermissions []model.RolePermission
	model.DB.Where("role_id = ?", roleID).Find(&rolePermissions)
	for _, p := range rolePermissions {
		permissions = append(permissions, p.PermissionCode)
	}

	return gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"phone":    user.Phone,
		"status":   user.Status,
		"role_id":  user.RoleID,
		"role":     role.Name,
	}, menuList, permissions, nil
}

// Logout 删除 Redis 中保存的 token
func (s *AuthService) Logout(token string) error {
	return cache.DeleteToken(token)
}
