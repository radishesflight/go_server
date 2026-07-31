// Package system roleMenu.go - 角色菜单权限分配 HTTP 入口
//
// 替代旧的 roleMenu.go(同包内,内容全重写)
//
// 接口列表:
//  GET  /api/system/roleMenu/allMenus              所有菜单(带 operations,供前端"分配菜单"对话框)
//  GET  /api/system/roleMenu/roleMenus?role_id=    某角色已分配的菜单 ID
//  GET  /api/system/roleMenu/roleOperations?role_id=  某角色已分配的操作(按 menu_id 分组)
//  POST /api/system/roleMenu/assign                分配 {role_id, menu_ids, operations}
//
// 业务码翻译表:
//  service.ErrRoleMenuRequireRole → CodeParamsInvalid
//  service.ErrRoleMenuAssign      → CodeUnknown
//  service.ErrRolePermAssign      → CodeUnknown
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// roleMenuSvc 角色-菜单-操作 业务入口
var roleMenuSvc = service.NewRoleMenuService()

// AssignMenuReq 分配请求体
// menu_ids: 角色可见的菜单 ID 列表
// operations: { menu_id: [operation_code, ...] }
type AssignMenuReq struct {
	RoleID     uint                `json:"role_id" binding:"required"`
	MenuIDs    []uint              `json:"menu_ids"`
	Operations map[uint][]string   `json:"operations"` // key 是 menu_id (uint),value 是 operation code 列表
}

// GetRoleMenuAllMenus 所有菜单(带 operations,供"分配菜单"对话框)
func GetRoleMenuAllMenus(c *gin.Context) {
	list := menuSvc.GetAllWithOperations()
	handler.Success(c, list)
}

// GetRoleMenuIDs 某角色已分配的菜单 ID
func GetRoleMenuIDs(c *gin.Context) {
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

	menuIDs, err := roleMenuSvc.GetRoleMenuIDs(uint(roleID))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"menu_ids": menuIDs,
	})
}

// GetRoleOperationCodes 某角色已分配的操作(按 menu_id 分组)
func GetRoleOperationCodes(c *gin.Context) {
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

	operations, err := roleMenuSvc.GetRoleOperationCodes(uint(roleID))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"operations": operations,
	})
}

// AssignMenusAndOperations 给角色分配菜单和操作
func AssignMenusAndOperations(c *gin.Context) {
	var req AssignMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	if err := roleMenuSvc.AssignMenusAndOperations(req.RoleID, req.MenuIDs, req.Operations); err != nil {
		switch {
		case errors.Is(err, service.ErrRoleMenuRequireRole):
			handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		case errors.Is(err, service.ErrRoleMenuAssign):
			handler.Error(c, handler.CodeUnknown, "分配菜单失败")
		case errors.Is(err, service.ErrRolePermAssign):
			handler.Error(c, handler.CodeUnknown, "分配操作失败")
		default:
			handler.Error(c, handler.CodeUnknown, "分配失败")
		}
		return
	}

	handler.Success(c, nil)
}
