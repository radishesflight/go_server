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

type TokenData struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	RoleID      uint     `json:"role_id"`
	Menus       []Menu   `json:"menus"`
	Permissions []string `json:"permissions"`
}

type Menu struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"`
}

func GenerateToken(userID uint, username string, roleID uint, menus []Menu, permissions []string) (string, error) {
	token := uuid.New().String()
	ctx := context.Background()

	menusJSON, _ := json.Marshal(menus)
	permissionsJSON, _ := json.Marshal(permissions)

	key := fmt.Sprintf("token:%s", token)
	if err := model.RDB.HSet(ctx, key, map[string]interface{}{
		"user_id":      userID,
		"username":     username,
		"role_id":      roleID,
		"menus":        string(menusJSON),
		"permissions":  string(permissionsJSON),
	}).Err(); err != nil {
		return "", err
	}

	if err := model.RDB.Expire(ctx, key, TokenExpire).Err(); err != nil {
		return "", err
	}

	return token, nil
}

func ValidateToken(token string) (*TokenData, error) {
	ctx := context.Background()
	key := fmt.Sprintf("token:%s", token)

	userID, err := model.RDB.HGet(ctx, key, "user_id").Uint64()
	if err != nil {
		return nil, err
	}

	username, _ := model.RDB.HGet(ctx, key, "username").Result()
	roleID, _ := model.RDB.HGet(ctx, key, "role_id").Uint64()
	menusStr, _ := model.RDB.HGet(ctx, key, "menus").Result()
	permissionsStr, _ := model.RDB.HGet(ctx, key, "permissions").Result()

	var menus []Menu
	var permissions []string
	json.Unmarshal([]byte(menusStr), &menus)
	json.Unmarshal([]byte(permissionsStr), &permissions)

	return &TokenData{
		UserID:      uint(userID),
		Username:    username,
		RoleID:      uint(roleID),
		Menus:       menus,
		Permissions: permissions,
	}, nil
}

func DeleteToken(token string) error {
	ctx := context.Background()
	key := fmt.Sprintf("token:%s", token)
	return model.RDB.Del(ctx, key).Err()
}

func UpdateTokenMenusAndPermissions(token string, menus []Menu, permissions []string) error {
	ctx := context.Background()
	key := fmt.Sprintf("token:%s", token)

	menusJSON, _ := json.Marshal(menus)
	permissionsJSON, _ := json.Marshal(permissions)

	model.RDB.HSet(ctx, key, map[string]interface{}{
		"menus":       string(menusJSON),
		"permissions": string(permissionsJSON),
	})
	return nil
}
