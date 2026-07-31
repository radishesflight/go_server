package middleware

import (
	"go_server/internal/handler"
	"go_server/pkg/cache"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			handler.Error(c, handler.CodeTokenMissing, "未携带令牌")
			c.Abort()
			return
		}

		tokenData, err := cache.ValidateToken(token)
		if err != nil {
			handler.Error(c, handler.CodeTokenInvalid, "令牌无效或已过期")
			c.Abort()
			return
		}

		c.Set("user_id", tokenData.UserID)
		c.Set("username", tokenData.Username)
		c.Set("role_id", tokenData.RoleID)
		c.Set("menus", tokenData.Menus)
		c.Set("permissions", tokenData.Permissions)

		c.Next()
	}
}
