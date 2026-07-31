// Package middleware permission.go - 权限校验中间件
//
// 直接匹配机制(替代旧的 URL 推断):
//  1. 登录时,auth_service.go 把角色关联的 (method, path) 拼成 ["GET /api/foo/:id", ...] 存 token
//  2. 请求进来时,直接拿 c.Request.Method + " " + c.FullPath() 查用户的 set
//  3. c.FullPath() 自动把 /api/system/adminRoles/123 解析回 /api/system/adminRoles/:id
//     (Gin 路由匹配后写入的)
//
// 优点:
//   - 不需要维护推断规则
//   - 同一个 path 不同 method 自动算两条权限(Gin 路由分别注册)
//   - 加新权限:在 admin_menu_operations 表加一行 + 关联角色,不改任何代码
//
// 公开路径(不走 PermissionMiddleware):
//
//	/api/login /api/logout
//	这些是单独注册在 /api 根组下,没挂这个中间件
package middleware

import (
	"go_server/internal/handler"

	"github.com/gin-gonic/gin"
)

// PermissionMiddleware 权限校验中间件
// 失败:返回 CodeNoPermission(2001) + Abort
// 成功:c.Next() 放行
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

		// 3. 取当前请求对应的路由模式
		//    c.FullPath() 返回注册的路由 pattern,例如 /api/system/adminRoles/:id
		//    对于没匹配上的路由(404),c.FullPath() 为空,放行(让后续 404 处理)
		fullPath := c.FullPath()
		if fullPath == "" {
			c.Next()
			return
		}

		// 4. 直接匹配 "METHOD /fullPath"
		key := c.Request.Method + " " + fullPath
		if !contains(permList, key) {
			handler.Error(c, handler.CodeNoPermission, "无权限")
			c.Abort()
			return
		}

		c.Next()
	}
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
