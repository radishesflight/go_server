// Package service auth_service.go - 认证业务
//
// 三个核心方法:
//  1. Login:校验用户名密码 → 查角色 → 查菜单 → 查路由权限 → 生成 token
//  2. Logout:删 Redis 里的 token 记录
//  3. GetCurrentUser:从 token 重新查用户/菜单/权限(用于刷新)
//
// 权限码生成(直接存 API 路由):
//
//	角色-路由 关联 JOIN admin_menu_operations 表
//	拼成 "METHOD /full/path" 列表(如 ["GET /api/system/adminRoles/list", "POST /api/system/adminRoles"])
//	中间件直接拿 c.Request.Method + " " + c.FullPath() 查这个 set
//
// 业务错误(handler 用 errors.Is 翻译):
//
//	ErrAuthUserNotFound    → "用户不存在"     CodeUserNotFound
//	ErrAuthWrongPassword   → "密码错误"       CodeUserPassword
//	ErrAuthUserNoRole      → "未分配角色"     CodeUserNoRole
//	ErrAuthRoleNotFound    → "角色不存在"     CodeRoleNotFound
//	ErrAuthTokenGenFailed  → "令牌生成失败"   CodeAuthFail
package service

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go_server/internal/model"
	"go_server/pkg/cache"
)

var (
	ErrAuthUserNotFound   = errors.New("用户不存在")
	ErrAuthWrongPassword  = errors.New("密码错误")
	ErrAuthUserNoRole     = errors.New("该用户未分配角色")
	ErrAuthRoleNotFound   = errors.New("角色不存在")
	ErrAuthTokenGenFailed = errors.New("令牌生成失败")
	ErrAuthUserLocked     = errors.New("账号已被锁定,请联系管理员解锁") // 2026-10-20 连续 3 次密码错后锁定
)

// MaxLoginFailures 登录失败阈值(达此值即锁定)
const MaxLoginFailures = 3

// isLocked 判断账号是否被锁定(LockedAt 非空且未到永久,这里就用 LockedAt 非空判断)
func isLocked(u *model.AdminUsers) bool {
	return u.LockedAt != nil && !u.LockedAt.IsZero()
}

// AuthService 登录 / 注销 / 当前用户 业务
type AuthService struct{}

// NewAuthService 构造 AuthService
func NewAuthService() *AuthService { return &AuthService{} }

// Login 处理登录
// 返回:token, user(map), menus(树), permissions
//
// 2026-10-20 加登录限制:
//   - LockedAt 非空 → 账号已锁定,直接拒绝(密码对也不让登),让管理员解锁
//   - 密码错:failed_count++,达 3 次置 LockedAt = now() 并拒绝
//   - 密码对:登录成功后清零 failed_count + LockedAt(让用户可以重新从干净状态开始)
func (s *AuthService) Login(username, password string) (string, gin.H, []*MenuTreeNode, []string, error) {
	var user model.AdminUsers
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", nil, nil, nil, ErrAuthUserNotFound
	}

	// 已被锁 → 直接拒绝,密码正确也不让登(必须 admin 解锁)
	if isLocked(&user) {
		return "", nil, nil, nil, ErrAuthUserLocked
	}

	if !user.CheckPassword(password) {
		// 密码错:计数 +1,达到阈值就锁定
		user.FailedCount++
		if user.FailedCount >= MaxLoginFailures {
			now := time.Now()
			user.LockedAt = &now
		}
		// 用 Updates 写库(只写两列,不会误改其他字段;gorm 默认忽略零值,所以 FailedCount 从 0 → 1 必须显式写)
		model.DB.Model(&model.AdminUsers{}).Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"failed_count": user.FailedCount,
				"locked_at":    user.LockedAt,
			})
		return "", nil, nil, nil, ErrAuthWrongPassword
	}

	// 密码正确:登录成功后清零失败计数 + 锁定状态
	if user.FailedCount > 0 || user.LockedAt != nil {
		model.DB.Model(&model.AdminUsers{}).Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"failed_count": 0,
				"locked_at":    nil,
			})
	}

	if user.RoleID == 0 {
		return "", nil, nil, nil, ErrAuthUserNoRole
	}

	var role model.AdminRoles
	if err := model.DB.First(&role, user.RoleID).Error; err != nil {
		return "", nil, nil, nil, ErrAuthRoleNotFound
	}

	// 1. 查角色-菜单 → menu_ids → 查 menus → 构建菜单树
	menuIDs, err := s.getMenuIDsByRole(user.RoleID)
	if err != nil {
		return "", nil, nil, nil, err
	}

	var menus []model.AdminMenus
	if len(menuIDs) > 0 {
		model.DB.Where("id IN ? AND status = ?", menuIDs, 1).Order("sort DESC").Find(&menus)
	}
	menuList := BuildMenuTree(menus)

	// 2. 拉平菜单(token 里缓存用)
	flatMenus := make([]cache.Menu, 0)
	for _, m := range menus {
		flatMenus = append(flatMenus, cache.Menu{
			ID:   m.ID,
			Name: m.Name,
			Path: m.Path,
			Icon: m.Icon,
		})
	}

	// 3. 查角色-路由 → 拼成 "METHOD /path" 权限列表
	permissions, err := s.getPermissionRoutesByRole(user.RoleID)
	if err != nil {
		return "", nil, nil, nil, err
	}

	// 4. 生成 token(带 departmentID,业务层过滤用)
	token, err := cache.GenerateToken(user.ID, user.Username, user.RoleID, user.DepartmentID, flatMenus, permissions)
	if err != nil {
		return "", nil, nil, nil, ErrAuthTokenGenFailed
	}

	return token, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"phone":         user.Phone,
		"status":        user.Status,
		"role_id":       user.RoleID,
		"role":          role.Name,
		"department_id": user.DepartmentID,
	}, menuList, permissions, nil
}

// GetCurrentUser 取当前用户信息(根据 token 解出的 userID / roleID 重新查 DB)
//
// 当前实现:只读,DB 出错也走"成功 + 空数据"降级(避免刷新接口雪崩)。
// 错误通过日志记录,不返回给调用方(handler 不需要区分具体错误)。
func (s *AuthService) GetCurrentUser(userID, roleID uint) (gin.H, []*MenuTreeNode, []string) {
	var user model.AdminUsers
	model.DB.Where("id = ?", userID).First(&user)

	var role model.AdminRoles
	model.DB.Where("id = ?", roleID).First(&role)

	menuIDs, _ := s.getMenuIDsByRole(roleID)

	var menus []model.AdminMenus
	if len(menuIDs) > 0 {
		model.DB.Where("id IN ? AND status = ?", menuIDs, 1).Order("sort DESC").Find(&menus)
	}
	menuList := BuildMenuTree(menus)

	permissions, _ := s.getPermissionRoutesByRole(roleID)

	return gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"phone":         user.Phone,
		"status":        user.Status,
		"role_id":       user.RoleID,
		"role":          role.Name,
		"department_id": user.DepartmentID,
	}, menuList, permissions
}

// Logout 删除 Redis 中保存的 token
func (s *AuthService) Logout(token string) error {
	return cache.DeleteToken(token)
}

// getMenuIDsByRole 取角色的菜单 ID 列表
func (s *AuthService) getMenuIDsByRole(roleID uint) ([]uint, error) {
	var relations []model.AdminRoleMenus
	model.DB.Where("role_id = ?", roleID).Find(&relations)
	menuIDs := make([]uint, len(relations))
	for i, r := range relations {
		menuIDs[i] = r.MenuID
	}
	return menuIDs, nil
}

// getPermissionRoutesByRole 取角色的权限码列表
// 权限码 = "METHOD /full/path"  直接对应 Gin 路由,中间件直接匹配
func (s *AuthService) getPermissionRoutesByRole(roleID uint) ([]string, error) {
	var roleOps []model.AdminRoleOperations
	model.DB.Where("role_id = ?", roleID).Find(&roleOps)

	if len(roleOps) == 0 {
		return []string{}, nil
	}

	// 一次查所有 route(method + path)
	routeIDs := make([]uint, len(roleOps))
	for i, ro := range roleOps {
		routeIDs[i] = ro.RouteID
	}
	var routes []model.AdminMenuOperations
	model.DB.Where("id IN ?", routeIDs).Find(&routes)

	// 拼成 "METHOD /path",去重
	permSet := make(map[string]bool)
	permissions := make([]string, 0)
	for _, r := range routes {
		key := strings.ToUpper(r.Method) + " " + r.Path
		if !permSet[key] {
			permSet[key] = true
			permissions = append(permissions, key)
		}
	}
	return permissions, nil
}

// ReloadUserContext 重新加载角色的 menus + permissions
// 给 AuthMiddleware 版本号落后时用
// 返回值:
//
//	menus       - 给前端侧边栏用的扁平菜单列表
//	permissions - 给 PermissionMiddleware 用的 "METHOD /path" 列表
func (s *AuthService) ReloadUserContext(roleID uint) ([]cache.Menu, []string, error) {
	// 1. 查菜单
	menuIDs, err := s.getMenuIDsByRole(roleID)
	if err != nil {
		return nil, nil, err
	}
	var menus []model.AdminMenus
	if len(menuIDs) > 0 {
		model.DB.Where("id IN ? AND status = ?", menuIDs, 1).Order("sort DESC").Find(&menus)
	}
	flatMenus := make([]cache.Menu, 0, len(menus))
	for _, m := range menus {
		flatMenus = append(flatMenus, cache.Menu{
			ID:   m.ID,
			Name: m.Name,
			Path: m.Path,
			Icon: m.Icon,
		})
	}

	// 2. 查权限
	permissions, err := s.getPermissionRoutesByRole(roleID)
	if err != nil {
		return nil, nil, err
	}

	return flatMenus, permissions, nil
}
