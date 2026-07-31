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

// AssignMenuReq 分配菜单 / 权限 请求体(JSON 字段不变,前端无感)
type AssignMenuReq struct {
	RoleID      uint     `json:"role_id" binding:"required"`
	MenuIDs     []uint   `json:"menu_ids"`
	Permissions []string `json:"permissions"`
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
