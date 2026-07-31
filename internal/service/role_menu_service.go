package service

import (
	"context"
	"errors"
	"strings"

	"go_server/internal/model"
	"go_server/pkg/cache"
)

// 业务错误(与原 handler 中的错误文案对应)
var (
	ErrRoleMenuRequireRole  = errors.New("请选择角色")
	ErrRoleMenuInvalidID    = errors.New("无效的角色ID")
	ErrRoleMenuAssign       = errors.New("分配菜单失败")
	ErrRolePermAssign       = errors.New("分配权限失败")
)

// RoleMenuService 角色-菜单 / 角色-权限 关联业务
type RoleMenuService struct{}

// NewRoleMenuService 构造 RoleMenuService
func NewRoleMenuService() *RoleMenuService { return &RoleMenuService{} }

// GetMenuIDsByRole 取某角色关联的菜单 ID 列表
func (s *RoleMenuService) GetMenuIDsByRole(roleID uint64) ([]uint, error) {
	if roleID == 0 {
		return nil, ErrRoleMenuRequireRole
	}
	var relations []model.RoleMenuRelation
	model.DB.Where("role_id = ?", roleID).Find(&relations)

	menuIDs := make([]uint, len(relations))
	for i, r := range relations {
		menuIDs[i] = r.MenuID
	}
	return menuIDs, nil
}

// GetMenuIDsAndNamesByRole 取某角色关联菜单 ID + 名称串(中文逗号分隔)
// 与原 GetMenusByRoleWithNames 行为一致
func (s *RoleMenuService) GetMenuIDsAndNamesByRole(roleID uint64) ([]uint, string, error) {
	if roleID == 0 {
		return nil, "", ErrRoleMenuRequireRole
	}

	var relations []model.RoleMenuRelation
	model.DB.Where("role_id = ?", roleID).Find(&relations)

	if len(relations) == 0 {
		return []uint{}, "", nil
	}

	menuIDs := make([]uint, len(relations))
	for i, r := range relations {
		menuIDs[i] = r.MenuID
	}

	var menus []model.AdminMenus
	model.DB.Where("id IN ?", menuIDs).Find(&menus)

	names := make([]string, len(menus))
	for i, m := range menus {
		names[i] = m.Name
	}
	return menuIDs, strings.Join(names, "，"), nil
}

// GetPermissionsByRole 取某角色的权限码列表
func (s *RoleMenuService) GetPermissionsByRole(roleID uint64) ([]string, error) {
	if roleID == 0 {
		return nil, ErrRoleMenuRequireRole
	}

	var rolePermissions []model.RolePermission
	model.DB.Where("role_id = ?", roleID).Find(&rolePermissions)

	permissions := make([]string, len(rolePermissions))
	for i, p := range rolePermissions {
		permissions[i] = p.PermissionCode
	}
	return permissions, nil
}

// AssignMenusToRole 给角色分配菜单和权限
// 业务行为与原 handler.AssignMenusToRole 完全一致:
//   1. 删除原 role_menu_relation
//   2. 批量创建新 role_menu_relation
//   3. 删除原 role_permission
//   4. 自动补充 view 权限(每个 menu 的 code:view)
//   5. 合并前端自定义 permission,去重
//   6. 批量创建新 role_permission
//   7. 异步更新使用该角色的用户 token 缓存
func (s *RoleMenuService) AssignMenusToRole(roleID uint, menuIDs []uint, permissions []string) error {
	if roleID == 0 {
		return ErrRoleMenuRequireRole
	}

	// 删除该角色的所有菜单关联
	model.DB.Where("role_id = ?", roleID).Delete(&model.RoleMenuRelation{})

	// 批量创建新的关联
	if len(menuIDs) > 0 {
		var relations []model.RoleMenuRelation
		for _, menuID := range menuIDs {
			relations = append(relations, model.RoleMenuRelation{
				RoleID: roleID,
				MenuID: menuID,
			})
		}
		if err := model.DB.Create(&relations).Error; err != nil {
			return ErrRoleMenuAssign
		}
	}

	// 删除该角色的所有权限关联
	model.DB.Where("role_id = ?", roleID).Delete(&model.RolePermission{})

	// 自动补充 view 权限:只要分配了菜单,就自动拥有该菜单的 view 权限
	allPermissions := make([]string, 0)
	if len(menuIDs) > 0 {
		var menus []model.AdminMenus
		model.DB.Where("id IN ?", menuIDs).Find(&menus)
		for _, menu := range menus {
			allPermissions = append(allPermissions, menu.Code+":view")
		}
	}

	// 合并前端传来的自定义权限
	if len(permissions) > 0 {
		allPermissions = append(allPermissions, permissions...)
	}

	// 去重
	permSet := make(map[string]bool)
	finalPermissions := make([]string, 0)
	for _, p := range allPermissions {
		if !permSet[p] {
			permSet[p] = true
			finalPermissions = append(finalPermissions, p)
		}
	}

	// 批量创建新的权限关联
	if len(finalPermissions) > 0 {
		var permissions []model.RolePermission
		for _, permCode := range finalPermissions {
			permissions = append(permissions, model.RolePermission{
				RoleID:         roleID,
				PermissionCode: permCode,
			})
		}
		if err := model.DB.Create(&permissions).Error; err != nil {
			return ErrRolePermAssign
		}
	}

	// 获取该角色的最新菜单(用于更新 token)
	var finalMenuIDs []uint
	if len(menuIDs) > 0 {
		finalMenuIDs = menuIDs
	} else {
		model.DB.Model(&model.RoleMenuRelation{}).Where("role_id = ?", roleID).Pluck("menu_id", &finalMenuIDs)
	}
	var menus []model.AdminMenus
	model.DB.Where("id IN ? AND status = ?", finalMenuIDs, 1).Find(&menus)
	menuList := make([]cache.Menu, len(menus))
	for i, m := range menus {
		menuList[i] = cache.Menu{ID: m.ID, Name: m.Name, Path: m.Path, Icon: m.Icon}
	}

	// 异步更新使用该角色的用户 token
	go updateTokensForRole(roleID, menuList, finalPermissions)

	return nil
}

// updateTokensForRole 异步:更新所有使用该角色的用户 token 缓存
// 行为与原 handler.updateTokensForRole 一致
func updateTokensForRole(roleID uint, menus []cache.Menu, permissions []string) {
	var users []model.AdminUsers
	model.DB.Where("role_id = ?", roleID).Find(&users)

	ctx := context.Background()

	for _, user := range users {
		pattern := "token:*"
		keys, _ := model.RDB.Keys(ctx, pattern).Result()

		var tokenKey string
		for _, key := range keys {
			userID, _ := model.RDB.HGet(ctx, key, "user_id").Uint64()
			if uint(userID) == user.ID {
				tokenKey = key
				break
			}
		}

		if tokenKey == "" {
			continue
		}

		token := strings.TrimPrefix(tokenKey, "token:")
		cache.UpdateTokenMenusAndPermissions(token, menus, permissions)
	}
}
