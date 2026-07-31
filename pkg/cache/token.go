package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go_server/internal/model"

	"github.com/google/uuid"
)

const TokenExpire = 24 * time.Hour

// TokenData token 解析出来的数据
//
// PermVersion 关键!
//
//	登录时记录当时的角色权限版本号
//	中间件每次请求跟 Redis 当前 version 比
//	落后 → 从 DB 重新加载,update token
//	这样改权限不用踢人重登录
type TokenData struct {
	UserID       uint     `json:"user_id"`
	Username     string   `json:"username"`
	RoleID       uint     `json:"role_id"`
	DataScope    int      `json:"data_scope"`
	DepartmentID uint     `json:"department_id"`
	Menus        []Menu   `json:"menus"`
	Permissions  []string `json:"permissions"`
	PermVersion  uint64   `json:"perm_version"`
}

type Menu struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"`
}

// GenerateToken 生成 token 并写 Redis
// 自动 init 角色权限版本号,记录到 token 里
func GenerateToken(userID uint, username string, roleID uint, dataScope int, departmentID uint, menus []Menu, permissions []string) (string, error) {
	token := uuid.New().String()
	ctx := context.Background()

	// 拿(并 init)当前角色的权限版本号
	permVersion := GetOrInitVersion(roleID)

	menusJSON, _ := json.Marshal(menus)
	permissionsJSON, _ := json.Marshal(permissions)

	key := fmt.Sprintf("token:%s", token)
	if err := model.RDB.HSet(ctx, key, map[string]interface{}{
		"user_id":       userID,
		"username":      username,
		"role_id":       roleID,
		"data_scope":    dataScope,
		"department_id": departmentID,
		"menus":         string(menusJSON),
		"permissions":   string(permissionsJSON),
		"perm_version":  permVersion,
	}).Err(); err != nil {
		return "", err
	}

	if err := model.RDB.Expire(ctx, key, TokenExpire).Err(); err != nil {
		return "", err
	}

	return token, nil
}

// ValidateToken 从 Redis 读 token
func ValidateToken(token string) (*TokenData, error) {
	ctx := context.Background()
	key := fmt.Sprintf("token:%s", token)

	userID, err := model.RDB.HGet(ctx, key, "user_id").Uint64()
	if err != nil {
		return nil, err
	}

	username, _ := model.RDB.HGet(ctx, key, "username").Result()
	roleID, _ := model.RDB.HGet(ctx, key, "role_id").Uint64()
	dataScope, _ := model.RDB.HGet(ctx, key, "data_scope").Int()
	departmentID, _ := model.RDB.HGet(ctx, key, "department_id").Uint64()
	menusStr, _ := model.RDB.HGet(ctx, key, "menus").Result()
	permissionsStr, _ := model.RDB.HGet(ctx, key, "permissions").Result()
	permVersion, _ := model.RDB.HGet(ctx, key, "perm_version").Uint64()

	var menus []Menu
	var permissions []string
	json.Unmarshal([]byte(menusStr), &menus)
	json.Unmarshal([]byte(permissionsStr), &permissions)

	return &TokenData{
		UserID:       uint(userID),
		Username:     username,
		RoleID:       uint(roleID),
		DataScope:    dataScope,
		DepartmentID: uint(departmentID),
		Menus:        menus,
		Permissions:  permissions,
		PermVersion:  permVersion,
	}, nil
}

// DeleteToken 删 token(登出用)
func DeleteToken(token string) error {
	ctx := context.Background()
	key := fmt.Sprintf("token:%s", token)
	return model.RDB.Del(ctx, key).Err()
}

// UpdateTokenMenusAndPermissions 更新 token 里的 menus + permissions + perm_version
// 中间件发现版本号落后时调用
func UpdateTokenMenusAndPermissions(token string, menus []Menu, permissions []string, permVersion uint64) error {
	ctx := context.Background()
	key := fmt.Sprintf("token:%s", token)

	menusJSON, _ := json.Marshal(menus)
	permissionsJSON, _ := json.Marshal(permissions)

	model.RDB.HSet(ctx, key, map[string]interface{}{
		"menus":        string(menusJSON),
		"permissions":  string(permissionsJSON),
		"perm_version": permVersion,
	})
	return nil
}
