package middleware

import (
	"go_server/config"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
// 从 config.CORS.AllowedOrigins 读取白名单,默认 "*"
// 支持格式:
//   - "*"           允许所有 origin
//   - "https://a.com,https://b.com" 逗号分隔白名单
//
// 行为:当 origin 在白名单内时,Access-Control-Allow-Origin 设为该 origin;
// 白名单为 "*" 时,直接返回 "*"。
func CORS() gin.HandlerFunc {
	origins := []string{"*"}
	if config.AppConfig != nil {
		origins = config.AppConfig.CORS.SplitOrigins()
	}
	allowAll := len(origins) == 1 && origins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowOrigin := "*"
		if !allowAll {
			for _, o := range origins {
				if o == origin {
					allowOrigin = origin
					break
				}
			}
			// 不在白名单中,不设置 Allow-Origin(浏览器会拒绝)
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
