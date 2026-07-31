// Package middleware permission.go - 权限校验中间件
//
// 用途:放在 AuthMiddleware 之后,校验当前用户对当前接口是否有权限
//
// 权限码规则:
//   menu_code + ":" + operation_code
//   例: "adminUsers:view" / "orders:import" / "users:batch_delete"
//
// 登录时由 auth_service.go 根据 admin_role_operations 表生成
// 这里只做**粗粒度 fallback 推断**(URL → 权限码),保证没显式分配也能用基础 CRUD
//
// HTTP method + URL path 推断规则:
//   GET    /api/system/{menu}[/{id}]              →  menu:view
//   POST   /api/system/{menu}                     →  menu:add
//   PUT    /api/system/{menu}/{id}                →  menu:edit
//   DELETE /api/system/{menu}/{id}                →  menu:delete
//   {ANY}   /api/system/{menu}/{op}               →  menu:op   (op 是子路径,如 import/export/batch_delete)
//
// 例外:
//   /api/login /api/logout /api/user/info /api/upload/image
//   这些是公开/已 AuthMiddleware 处理的,不需要 PermissionMiddleware
//
// 加新权限码的流程:
//   1. 后端:在 admin_menu_operations 表加一条操作(不需要改代码)
//   2. 前端:前端自动从后端 dynamic 拿 operations,不用改代码
package middleware

import (
	"strconv"
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

	// 公开/特殊路径不参与推断(由路由组决定)
	if isPublicPath(path) {
		return ""
	}

	path = strings.TrimPrefix(path, "/api/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}

	// 路由 /api/{group}/{menu}[/{id}][/...]
	// parts[0] = "system"  (因为 /api/system/...)
	// parts[1] = menu code
	menu := parts[1]
	if menu == "system" && len(parts) >= 3 {
		menu = parts[2]
	}

	// 标准 CRUD pattern:
	//   /api/system/{menu}                  GET/POST
	//   /api/system/{menu}/{id}             GET/PUT/DELETE
	if len(parts) <= 4 && isStandardCRUD(parts) {
		methodMap := map[string]string{
			"GET":    "view",
			"POST":   "add",
			"PUT":    "edit",
			"DELETE": "delete",
		}
		if op, ok := methodMap[method]; ok {
			return menu + ":" + op
		}
	}

	// 特殊操作 pattern:
	//   /api/system/{menu}/{op}[/...]       method 通常是 POST/GET
	//   例: POST /api/system/adminUsers/import → adminUsers:import
	if len(parts) >= 4 {
		op := parts[3]
		if op != "" && !isNumeric(op) {
			return menu + ":" + op
		}
	}

	return ""
}

// isStandardCRUD 判断是否是标准 CRUD pattern
//   /api/system/{menu}                  → 1 段
//   /api/system/{menu}/{id}             → 2 段(ID)
func isStandardCRUD(parts []string) bool {
	// parts[0] = "system"
	// parts[1] = menu
	// parts[2] = id 或空(POST 时)
	// parts[3] = ?
	if len(parts) == 3 {
		// /api/system/{menu}  → POST 时 add,GET 时 view
		return true
	}
	if len(parts) == 4 && isNumeric(parts[3]) {
		// /api/system/{menu}/{id}  → GET/PUT/DELETE
		return true
	}
	return false
}

// isNumeric 判断字符串是否是数字(ID)
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// isPublicPath 公开路径不参与权限推断
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/api/login",
		"/api/logout",
		"/api/user/info",
		"/api/upload/image",
	}
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	return false
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
