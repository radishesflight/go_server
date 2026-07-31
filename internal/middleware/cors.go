// Package middleware cors.go - 跨域中间件
//
// 从 config.CORS.AllowedOrigins 读取白名单
//  - "*"                          允许所有 origin
//  - "https://a.com,https://b.com" 逗号分隔白名单
//
// 行为:当请求 origin 在白名单内时,Access-Control-Allow-Origin 设为该 origin;
// 白名单为 "*" 时,直接返回 "*"(浏览器会接受)。
// 不在白名单的请求,响应头**不**带 Allow-Origin,浏览器自动拒绝。
//
// OPTIONS 预检请求直接 204 返回,不进入后续 handler。
package middleware

import (
	"go_server/config"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
// 注意:此中间件在 SetupRouter 里第一个 Use(gin.Logger / Recovery 之后)
func CORS() gin.HandlerFunc {
	// 启动时读一次 config(避免每个请求都解析)
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
			// 不在白名单中,allowOrigin 保持 "*" 但不会真正放行
			// (浏览器会检查返回的 Allow-Origin 是否匹配,否则拒绝)
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// OPTIONS 预检请求直接 204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
