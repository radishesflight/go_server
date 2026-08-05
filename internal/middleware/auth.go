// Package middleware auth.go - 鉴权中间件
//
// 用途:放在需要登录的路由组上,校验请求是否携带有效 token
//
// 流程:
//  1. 读 Header "Authorization" 拿 token
//  2. 调 cache.ValidateToken 验证(token 存 Redis Hash)
//  3. **版本号检查**(关键!):
//     - 拿 Redis 里的 role_perm_version:<roleID> 跟 token 里的 PermVersion 比
//     - 落后 → 调 service 重新查 menus + permissions,update token
//     - 不落后 → 直接用 token 里的(快)
//  4. 把 user_id / username / role_id / menus / permissions 塞到 gin.Context
//  5. 调 c.Next() 放行;失败 c.Abort() 截断
//
// 错误处理走业务码(CodeTokenMissing / CodeTokenInvalid),
// 而不是 HTTP 401(后端业务码模式下 HTTP 恒 200)
package middleware

import (
	"go_server/internal/handler"
	"go_server/internal/service"
	"go_server/pkg/cache"
	"go_server/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// authSvc 懒加载权限时用
var authSvc = service.NewAuthService()

// AuthMiddleware 鉴权中间件
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

		// 3. 版本号检查 - 关键!
		//    登录时记录 version,改权限后 INCR
		//    落后就重新加载,不落后就直接用 token 缓存(快)
		menus := tokenData.Menus
		permissions := tokenData.Permissions
		currentVersion := cache.GetOrInitVersion(tokenData.RoleID)

		if tokenData.PermVersion < currentVersion {
			// 权限被改了,重新查 DB
			newMenus, newPerms, reloadErr := authSvc.ReloadUserContext(tokenData.RoleID)
			if reloadErr == nil {
				menus = newMenus
				permissions = newPerms
				// 同步更新 token(下次就不需要再 reload 了)
				cache.UpdateTokenMenusAndPermissions(token, menus, permissions, currentVersion)
				logger.L.Info("perm cache reloaded (lazy)",
					zap.Uint("user_id", tokenData.UserID),
					zap.Uint("role_id", tokenData.RoleID),
					zap.Uint64("token_version", tokenData.PermVersion),
					zap.Uint64("current_version", currentVersion),
					zap.Int("menus", len(menus)),
					zap.Int("permissions", len(permissions)),
					zap.String("path", c.FullPath()),
				)
			} else {
				// reload 失败 → 降级用旧值,不阻断请求
				logger.L.Warn("perm cache reload failed, fallback to token cache",
					zap.Uint("user_id", tokenData.UserID),
					zap.Uint("role_id", tokenData.RoleID),
					zap.Uint64("token_version", tokenData.PermVersion),
					zap.Uint64("current_version", currentVersion),
					zap.Error(reloadErr),
				)
			}
		}

		// 4. 注入上下文(供后续 handler 使用)
		c.Set("user_id", tokenData.UserID)
		c.Set("username", tokenData.Username)
		c.Set("role_id", tokenData.RoleID)
		c.Set("data_scope", tokenData.DataScope)
		c.Set("department_id", tokenData.DepartmentID)
		c.Set("menus", menus)
		c.Set("permissions", permissions)

		// 5. 放行
		c.Next()
	}
}
