package middleware

import (
	"strings"

	"go_server/internal/handler"

	"github.com/gin-gonic/gin"
)

func PermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("permissions")
		if !exists {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		permList := permissions.([]string)
		if len(permList) == 0 {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		permission := inferPermission(c.Request.Method, c.Request.URL.Path)

		if permission != "" && !contains(permList, permission) {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

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

func contains(arr []string, item string) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}
