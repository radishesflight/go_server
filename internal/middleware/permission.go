// Package middleware permission.go - 权限校验中间件
//
// 用途:放在 AuthMiddleware 之后,校验当前用户对当前接口是否有权限
//
// 权限码生成规则(由 inferPermission):
//   /api/system/adminUsers      POST  →  "adminUsers:add"
//   /api/system/adminUsers/:id  GET   →  "adminUsers:view"
//   /api/system/adminUsers/:id  PUT   →  "adminUsers:edit"
//   /api/system/adminUsers/:id  DELETE → "adminUsers:delete"
//
// HTTP method → 操作名:
//   GET    → view
//   POST   → add
//   PUT    → edit
//   DELETE → delete
//
// URL 解析:
//   /api/{group}/{menu}[/{id}][/...]
//   例子:/api/system/adminUsers/list
//     parts[1] = "system"   ← 第二段
//     parts[2] = "adminUsers" ← 菜单标识(取这个)
//   例子:/api/system/adminMenus/options
//     parts[1] = "system"
//     parts[2] = "adminMenus"
//
// 必须放在 AuthMiddleware 之后(因为要从 gin.Context 读 permissions)
package middleware

import (
	"strings"

	"go_server/internal/handler"

	"github.com/gin-gonic/gin"
)

// PermissionMiddleware 权限校验中间件
// 失败时返回 CodeNoPermission(2001) + Abort
// 成功时 c.Next() 放行
func PermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 gin.Context 拿 permissions(由 AuthMiddleware 设置)
		permissions, exists := c.Get("permissions")
		if !exists {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		// 2. 没分配任何权限
		permList := permissions.([]string)
		if len(permList) == 0 {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		// 3. 推断当前请求需要的权限码
		permission := inferPermission(c.Request.Method, c.Request.URL.Path)

		// 4. 比对(空 permission 表示方法不识别,放行)
		if permission != "" && !contains(permList, permission) {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// inferPermission 根据 HTTP method + URL path 推断权限码
// 返回 "menu:operation" 格式,无法推断返回 ""
func inferPermission(method, path string) string {
	method = strings.ToUpper(method)

	methodMap := map[string]string{
		"GET":    "view",
		"POST":   "add",
		"PUT":    "edit",
		"DELETE": "delete",
	}

	operation, ok := methodMap[method]
	if !ok {
		return ""
	}

	path = strings.TrimPrefix(path, "/api/")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}

	// URL结构: /system/adminUsers/list 或 /system/adminMenus/1
	// parts[1] = "system" 是固定的
	// parts[2] = 菜单标识 (adminUsers, adminMenus, adminRoles, roleMenu)
	menu := parts[1]
	if menu == "system" && len(parts) >= 3 {
		menu = parts[2]
	}

	return menu + ":" + operation
}

// contains 简单的 slice contains
func contains(arr []string, item string) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}
