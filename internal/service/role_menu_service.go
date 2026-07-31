// Package service role_menu_service.go - 角色-菜单-操作 业务
//
// 核心:AssignMenusAndOperations
//  1. 删 admin_role_menus 旧记录
//  2. 批量 insert 新菜单关联
//  3. 删 admin_role_operations 旧记录
//  4. 查 menu.code + operation.code 拼成权限码
//  5. 批量 insert 新 role_operations
//  6. 异步删除该角色所有用户的 token(让他们重新登录拿新权限)
//
// 业务错误:
//  ErrRoleMenuRequireRole → role_id 缺失或无效
//  ErrRoleMenuAssign      → 关联菜单失败
//  ErrRolePermAssign      → 关联操作失败
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go_server/internal/model"
	"go_server/pkg/cache"
)

var (
	ErrRoleMenuRequireRole = errors.New("请选择角色")
	ErrRoleMenuInvalidID   = errors.New("无效的角色ID")
	ErrRoleMenuAssign      = errors.New("分配菜单失败")
	ErrRolePermAssign      = errors.New("分配操作失败")
)

// RoleMenuService 角色-菜单-操作 业务
type RoleMenuService struct{}

// NewRoleMenuService 构造 RoleMenuService
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

// GetRoleOperationCodes 角色已分配的操作,按 menu_id 分组
// 返回值:map[menu_id][]operation_code
// 给前端"分配菜单"对话框初始化用
func (s *RoleMenuService) GetRoleOperationCodes(roleID uint) (map[uint][]string, error) {
	if roleID == 0 {
		return nil, ErrRoleMenuRequireRole
	}

	var roleOps []model.AdminRoleOperations
	model.DB.Where("role_id = ?", roleID).Find(&roleOps)

	if len(roleOps) == 0 {
		return map[uint][]string{}, nil
	}

	// 一次查所有 operation
	opIDs := make([]uint, len(roleOps))
	for i, ro := range roleOps {
		opIDs[i] = ro.OperationID
	}

	var ops []model.AdminMenuOperations
	model.DB.Where("id IN ?", opIDs).Find(&ops)

	opMap := make(map[uint]model.AdminMenuOperations)
	for _, op := range ops {
		opMap[op.ID] = op
	}

	result := make(map[uint][]string)
	for _, ro := range roleOps {
		if op, ok := opMap[ro.OperationID]; ok {
			result[ro.MenuID] = append(result[ro.MenuID], op.Code)
		}
	}
	return result, nil
}

// AssignMenusAndOperations 给角色分配菜单和操作
// menuIDs: 角色可见的菜单 ID 列表
// operations: { menu_id: [operation_code, ...] } 角色对每个菜单的哪些操作有权限
func (s *RoleMenuService) AssignMenusAndOperations(roleID uint, menuIDs []uint, operations map[uint][]string) error {
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

	// 4. 查所有 menu_ids 对应的 operations(code + id)
	var allOps []model.AdminMenuOperations
	if len(menuIDs) > 0 {
		model.DB.Where("menu_id IN ?", menuIDs).Find(&allOps)
	}
	// key: "menu_id:op_code" → op_id
	opCodeToID := make(map[string]uint)
	for _, op := range allOps {
		opCodeToID[fmt.Sprintf("%d:%s", op.MenuID, op.Code)] = op.ID
	}

	// 5. 批量 insert 新 role_operations
	var roleOps []model.AdminRoleOperations
	for menuID, opCodes := range operations {
		for _, code := range opCodes {
			if opID, ok := opCodeToID[fmt.Sprintf("%d:%s", menuID, code)]; ok {
				roleOps = append(roleOps, model.AdminRoleOperations{
					RoleID:      roleID,
					MenuID:      menuID,
					OperationID: opID,
				})
			}
		}
	}
	if len(roleOps) > 0 {
		if err := model.DB.Create(&roleOps).Error; err != nil {
			return ErrRolePermAssign
		}
	}

	// 6. 异步清掉该角色所有用户的 token(强制重新登录,拿最新 permissions)
	go invalidateTokensForRole(roleID)

	return nil
}

// invalidateTokensForRole 异步:删除使用该角色的所有用户的 token
// 简化策略:删 token → 下次请求 token 失效 → 自动跳登录
// (原版用 updateTokensForRole 重新刷 token,但代码复杂,这里直接删)
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

// 避免 unused import 警告(原版用的 strings/cache 在新版本里改用其他方式)
var _ = strings.TrimPrefix
var _ = cache.UpdateTokenMenusAndPermissions
