// Package system roleMenu.go - 角色菜单权限分配 HTTP 入口
//
// 替代旧的 roleMenu.go(同包内,内容全重写)
//
// 接口列表:
//
//	GET  /api/system/roleMenu/allMenus            所有菜单(带 routes,供前端"分配菜单"对话框)
//	GET  /api/system/roleMenu/roleMenus?role_id=  某角色已分配的菜单 ID
//	GET  /api/system/roleMenu/roleRoutes?role_id= 某角色已分配的路由 ID
//	PUT  /api/system/roleMenu/assign              分配 {role_id, menu_ids, route_ids}
//
// 业务码翻译表:
//
//	service.ErrRoleMenuRequireRole → CodeParamsInvalid
//	service.ErrRoleMenuAssign      → CodeUnknown
//	service.ErrRolePermAssign      → CodeUnknown
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// roleMenuSvc 角色-菜单-路由 业务入口
var roleMenuSvc = service.NewRoleMenuService()

// AssignMenuReq 分配请求体
// route_ids 是 admin_menu_operations.id 的列表
type AssignMenuReq struct {
	RoleID   uint   `json:"role_id" binding:"required"`
	MenuIDs  []uint `json:"menu_ids"`
	RouteIDs []uint `json:"route_ids"`
}

// GetRoleMenuAllMenus 所有菜单(带 routes,供"分配菜单"对话框)
func GetRoleMenuAllMenus(c *gin.Context) {
	list := menuSvc.GetAllWithOperations()
	handler.Success(c, list)
}

// GetRoleMenuIDs 某角色已分配的菜单 ID
func GetRoleMenuIDs(c *gin.Context) {
	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	menuIDs, err := roleMenuSvc.GetRoleMenuIDs(roleID)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"menu_ids": menuIDs,
	})
}

// GetRoleRouteIDs 某角色已分配的路由 ID
func GetRoleRouteIDs(c *gin.Context) {
	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}

	routeIDs, err := roleMenuSvc.GetRoleRouteIDs(roleID)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return
	}

	handler.Success(c, gin.H{
		"route_ids": routeIDs,
	})
}

// AssignMenusAndOperations 给角色分配菜单和路由
func AssignMenusAndOperations(c *gin.Context) {
	var req AssignMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	if err := roleMenuSvc.AssignMenusAndOperations(req.RoleID, req.MenuIDs, req.RouteIDs); err != nil {
		switch {
		case errors.Is(err, service.ErrRoleMenuRequireRole):
			handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		case errors.Is(err, service.ErrRoleMenuAssign):
			handler.Error(c, handler.CodeUnknown, "分配菜单失败")
		case errors.Is(err, service.ErrRolePermAssign):
			handler.Error(c, handler.CodeUnknown, "分配路由失败")
		default:
			handler.Error(c, handler.CodeUnknown, "分配失败")
		}
		return
	}

	handler.Success(c, nil)
}

// parseRoleID 解析 query 里的 role_id,失败直接写错误响应
func parseRoleID(c *gin.Context) (uint, bool) {
	roleIDStr := c.Query("role_id")
	if roleIDStr == "" {
		handler.Error(c, handler.CodeParamsInvalid, "请选择角色")
		return 0, false
	}
	roleID, err := strconv.ParseUint(roleIDStr, 10, 64)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的角色ID")
		return 0, false
	}
	return uint(roleID), true
}
