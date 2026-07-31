// Package system roleMenu.go - 角色-菜单 / 角色-权限 HTTP 入口
//
// 这些接口是"角色管理"页里"分配菜单"对话框用的:
//  GET  /api/system/adminRoles/roleMenus            查某角色已分配的菜单 ID
//  GET  /api/system/adminRoles/roleMenusWithNames   查菜单名(用中文逗号拼成串)
//  GET  /api/system/adminRoles/rolePermissions      查某角色的权限码列表
//  POST /api/system/adminRoles/roleMenus            提交分配(删除老的 + 写新的)
//
// 业务码翻译表:
//  service.ErrRoleMenuAssign → CodeUnknown  (9999)
//  service.ErrRolePermAssign → CodeUnknown  (9999)
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// roleMenuSvc 角色-菜单 / 角色-权限 业务入口
var roleMenuSvc = service.NewRoleMenuService()

// AssignMenuReq 分配请求体
// 后端会:
//  1. 删 role_menu_relation 旧记录
//  2. 批量 insert 新记录(menu_ids)
//  3. 自动补充 <menu_code>:view 权限(每个 menu 默认有 view 权限)
//  4. 合并前端传的 permissions 字段(按钮级权限)
//  5. 异步刷新使用该角色的用户 token 缓存
type AssignMenuReq struct {
	RoleID      uint     `json:"role_id" binding:"required"`
	MenuIDs     []uint   `json:"menu_ids"`     // 菜单 ID 列表
	Permissions []string `json:"permissions"` // 自定义权限码(如 "adminUsers:add")
}

// GetMenusByRole 取某角色关联的菜单 ID 列表
func GetMenusByRole(c *gin.Context) {
	roleIDStr := c.Query("role_id")
	if roleIDStr == "" {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 64)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	menuIDs, err := roleMenuSvc.GetMenuIDsByRole(roleID)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"menu_ids": menuIDs,
	})
}

// GetMenusByRoleWithNames 取某角色关联菜单 ID + 名称串
func GetMenusByRoleWithNames(c *gin.Context) {
	roleIDStr := c.Query("role_id")
	if roleIDStr == "" {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 64)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	menuIDs, names, err := roleMenuSvc.GetMenuIDsAndNamesByRole(roleID)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"menu_ids":   menuIDs,
		"menu_names": names,
	})
}

// GetPermissionsByRole 取某角色权限码列表
func GetPermissionsByRole(c *gin.Context) {
	roleIDStr := c.Query("role_id")
	if roleIDStr == "" {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 64)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return
	}

	permissions, err := roleMenuSvc.GetPermissionsByRole(roleID)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"permissions": permissions,
	})
}

// AssignMenusToRole 给角色分配菜单和权限
func AssignMenusToRole(c *gin.Context) {
	var req AssignMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	err := roleMenuSvc.AssignMenusToRole(req.RoleID, req.MenuIDs, req.Permissions)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleMenuAssign):
			handler.Error(c, handler.CodeUnknown, "分配菜单失败")
		case errors.Is(err, service.ErrRolePermAssign):
			handler.Error(c, handler.CodeUnknown, "分配权限失败")
		default:
			handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		}
		return
	}

	handler.Success(c, nil)
}
