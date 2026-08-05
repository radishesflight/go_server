// Package cache perm_version.go - 角色权限版本号
//
// 用途:权限改了不踢人,只 bump 一个版本号,中间件下次请求发现 token 落后就重新加载
//
// Redis key: role_perm_version:<roleID>  (string,无过期)
//
//	值是 uint64,每次 AssignMenusAndOperations 成功后 INCR
//	首次访问(GET)如果不存在,SET 为 0(Init)
//
// 工作流:
//
//	登录:  GetOrInitVersion(roleID) → 拿到当前 version,塞到 token 里
//	改权限: BumpVersion(roleID)     → INCR,所有该角色 token 都"过期了"
//	请求:  AuthMiddleware 比较 token.PermVersion vs currentVersion
//	       落后就调 service 重新查,update token,继续处理
//
// 优点:
//   - 不需要踢人,用户无感知
//   - 没有竞态:版本号单调递增,token version < current version 必重载
package cache

import (
	"context"
	"fmt"

	"go_server/internal/model"
)

// rolePermVersionKey 返回 Redis key
// 例: "role_perm_version:1"
func rolePermVersionKey(roleID uint) string {
	return fmt.Sprintf("role_perm_version:%d", roleID)
}

// GetOrInitVersion 拿角色的当前权限版本号
// 如果 Redis 里没这个 key,初始化为 0 再返回
func GetOrInitVersion(roleID uint) uint64 {
	ctx := context.Background()
	key := rolePermVersionKey(roleID)

	// 先 GET(没拿到是 nil)
	v, err := model.RDB.Get(ctx, key).Uint64()
	if err == nil {
		return v
	}

	// 没拿到 → SETNX 初始化为 0
	// 多个请求并发初始化也只会成功一次(SetNX 原子)
	model.RDB.SetNX(ctx, key, 0, 0)
	v2, _ := model.RDB.Get(ctx, key).Uint64()
	return v2
}

// BumpVersion 给角色权限版本号 +1
// 调 AssignMenusAndOperations 后调用
func BumpVersion(roleID uint) {
	ctx := context.Background()
	key := rolePermVersionKey(roleID)
	model.RDB.Incr(ctx, key)
}
