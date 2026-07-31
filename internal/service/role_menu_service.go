// Package service role_menu_service.go - 角色-菜单-路由 业务
//
// 核心:AssignMenusAndOperations
//  1. 删 admin_role_menus 旧记录
//  2. 批量 insert 新菜单关联
//  3. 删 admin_role_operations 旧记录
//  4. 批量 insert 新 role_operations(route_id)
//  5. 异步删除该角色所有用户的 token(让他们重新登录拿新权限)
//
// 新设计:不再拼"menu:op"权限码,只存路由 ID(route_id)
//
//	admin_menu_operations  表本身就是权限定义表(method + path)
//	admin_role_operations  关联角色到具体路由
//
// 业务错误:
//
//	ErrRoleMenuRequireRole → role_id 缺失或无效
//	ErrRoleMenuAssign      → 关联菜单失败
//	ErrRolePermAssign      → 关联路由失败
package service

import (
	"context"
	"errors"
	"fmt"

	"go_server/internal/model"
)

var (
	ErrRoleMenuRequireRole = errors.New("请选择角色")
	ErrRoleMenuInvalidID   = errors.New("无效的角色ID")
	ErrRoleMenuAssign      = errors.New("分配菜单失败")
	ErrRolePermAssign      = errors.New("分配路由失败")
)

// RoleMenuService 角色-菜单-路由 业务
type RoleMenuService struct{}

// NewRoleMenuService 构造
func NewRoleMenuService() *RoleMenuService { return &RoleMenuService{} }

// GetRoleMenuIDs 取某角色已分配的菜单 ID 列表
func (s *RoleMenuService) GetRoleMenuIDs(roleID uint) ([]uint, error) {
	if roleID == 0 {
		return nil, ErrRoleMenuRequireRole
	}
	var relations []model.AdminRoleMenus
	model.DB.Where("role_id = ?", roleID).Find(&relations)
	menuIDs := make([]uint, len(relations))
	for i, r := range relations {
		menuIDs[i] = r.MenuID
	}
	return menuIDs, nil
}

// GetRoleRouteIDs 取某角色已分配的路由 ID 列表
// 给"权限分配"页面回显用
func (s *RoleMenuService) GetRoleRouteIDs(roleID uint) ([]uint, error) {
	if roleID == 0 {
		return nil, ErrRoleMenuRequireRole
	}
	var relations []model.AdminRoleOperations
	model.DB.Where("role_id = ?", roleID).Find(&relations)
	routeIDs := make([]uint, len(relations))
	for i, r := range relations {
		routeIDs[i] = r.RouteID
	}
	return routeIDs, nil
}

// AssignMenusAndOperations 给角色分配菜单和路由
// menuIDs:  角色可见的菜单 ID 列表
// routeIDs: 角色可调用的路由 ID 列表(来自 admin_menu_operations.id)
func (s *RoleMenuService) AssignMenusAndOperations(roleID uint, menuIDs []uint, routeIDs []uint) error {
	if roleID == 0 {
		return ErrRoleMenuRequireRole
	}

	// 1. 删 admin_role_menus 旧记录
	if err := model.DB.Where("role_id = ?", roleID).Delete(&model.AdminRoleMenus{}).Error; err != nil {
		return ErrRoleMenuAssign
	}

	// 2. 批量 insert 新菜单关联
	if len(menuIDs) > 0 {
		var relations []model.AdminRoleMenus
		for _, mid := range menuIDs {
			relations = append(relations, model.AdminRoleMenus{
				RoleID: roleID,
				MenuID: mid,
			})
		}
		if err := model.DB.Create(&relations).Error; err != nil {
			return ErrRoleMenuAssign
		}
	}

	// 3. 删 admin_role_operations 旧记录
	if err := model.DB.Where("role_id = ?", roleID).Delete(&model.AdminRoleOperations{}).Error; err != nil {
		return ErrRolePermAssign
	}

	// 4. 批量 insert 新 role_operations(去重)
	if len(routeIDs) > 0 {
		seen := make(map[uint]bool, len(routeIDs))
		var roleOps []model.AdminRoleOperations
		for _, rid := range routeIDs {
			if seen[rid] || rid == 0 {
				continue
			}
			seen[rid] = true
			roleOps = append(roleOps, model.AdminRoleOperations{
				RoleID:  roleID,
				RouteID: rid,
			})
		}
		if len(roleOps) > 0 {
			if err := model.DB.Create(&roleOps).Error; err != nil {
				return ErrRolePermAssign
			}
		}
	}

	// 5. 异步清掉该角色所有用户的 token(强制重新登录,拿最新 permissions)
	go invalidateTokensForRole(roleID)

	return nil
}

// invalidateTokensForRole 异步:删除使用该角色的所有用户的 token
// 简化策略:删 token → 下次请求 token 失效 → 自动跳登录
func invalidateTokensForRole(roleID uint) {
	var users []model.AdminUsers
	model.DB.Where("role_id = ?", roleID).Find(&users)

	ctx := context.Background()
	for _, user := range users {
		pattern := "token:*"
		keys, _ := model.RDB.Keys(ctx, pattern).Result()
		for _, key := range keys {
			uid, _ := model.RDB.HGet(ctx, key, "user_id").Uint64()
			if uint(uid) == user.ID {
				model.RDB.Del(ctx, key)
				break
			}
		}
	}
}

// 防 unused 警告(fmt 在新设计里没用,留接口以后报错信息可能用)
var _ = fmt.Sprintf
