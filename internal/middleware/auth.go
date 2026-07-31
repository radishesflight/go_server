// Package middleware auth.go - 鉴权中间件
//
// 用途:放在需要登录的路由组上,校验请求是否携带有效 token
//
// 流程:
//  1. 读 Header "Authorization" 拿 token
//  2. 调 cache.ValidateToken 验证(token 存 Redis Hash)
//  3. 把 user_id / username / role_id / menus / permissions 塞到 gin.Context
//  4. 调 c.Next() 放行;失败 c.Abort() 截断
//
// 错误处理走业务码(CodeTokenMissing / CodeTokenInvalid),
// 而不是 HTTP 401(后端业务码模式下 HTTP 恒 200)
package middleware

import (
	"go_server/internal/handler"
	"go_server/pkg/cache"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 鉴权中间件
// 失败时:直接返回业务码错误 + Abort,handler 不会再执行
// 成功时:把用户信息写入 gin.Context,后续 handler 可读
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 取 token
		token := c.GetHeader("Authorization")
		if token == "" {
			handler.Error(c, handler.CodeTokenMissing, "未携带令牌")
			c.Abort()
			return
		}

		// 2. 验证 token(从 Redis 读)
		tokenData, err := cache.ValidateToken(token)
		if err != nil {
			handler.Error(c, handler.CodeTokenInvalid, "令牌无效或已过期")
			c.Abort()
			return
		}

		// 3. 注入上下文(供后续 handler 使用)
		//    user_id / role_id:GetCurrentUser 用
		//    menus / permissions:PermissionMiddleware 用
		c.Set("user_id", tokenData.UserID)
		c.Set("username", tokenData.Username)
		c.Set("role_id", tokenData.RoleID)
		c.Set("menus", tokenData.Menus)
		c.Set("permissions", tokenData.Permissions)

		// 4. 放行
		c.Next()
	}
}
