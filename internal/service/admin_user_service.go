// Package service admin_user_service.go - 用户管理业务
//
// CRUD + 软删除;密码用 bcrypt 加密(在 SetPassword / CheckPassword 里)
// 业务错误:
//
//	ErrUserInvalidID       → 无效的 ID
//	ErrUserNotFound        → 用户不存在
//	ErrUserNameDuplicate   → 用户名已存在
//	ErrUserPasswordHash    → 密码加密失败
//	ErrUserCreate/Update/Delete → DB 操作失败
package service

import (
	"errors"

	"go_server/internal/model"
)

// 业务错误(与原 handler 中的错误文案对应)
var (
	ErrUserInvalidID     = errors.New("无效的用户ID")
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserNameDuplicate = errors.New("用户名已存在")
	ErrUserPasswordHash  = errors.New("密码加密失败")
	ErrUserPasswordWrong = errors.New("当前密码错误")
	ErrUserCreate        = errors.New("创建用户失败")
	ErrUserUpdate        = errors.New("更新用户失败")
	ErrUserDelete        = errors.New("删除用户失败")
)

// AdminUserService 用户管理业务
type AdminUserService struct{}

// NewAdminUserService 构造 AdminUserService
func NewAdminUserService() *AdminUserService { return &AdminUserService{} }

// formatUser 把用户 model 序列化成给前端的 map
// 字段名 / 值与原 handler.formatUser 完全一致
func formatUser(user model.AdminUsers) map[string]interface{} {
	var roleName string
	if user.RoleID > 0 {
		var role model.AdminRoles
		if err := model.DB.Where("id = ?", user.RoleID).First(&role).Error; err == nil {
			roleName = role.Name
		}
	}

	return map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"phone":      user.Phone,
		"status":     user.Status,
		"role_id":    user.RoleID,
		"role":       roleName,
		"created_at": model.FormatTime(user.CreatedAt),
		"updated_at": model.FormatTime(user.UpdatedAt),
	}
}

// GetList 分页查询用户列表
// 返回:list, total
func (s *AdminUserService) GetList(page, size, status int) ([]map[string]interface{}, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var users []model.AdminUsers
	var total int64

	db := model.DB.Model(&model.AdminUsers{})
	if status > 0 {
		db = db.Where("status = ?", status)
	}

	db.Count(&total)
	db.Offset((page - 1) * size).Limit(size).Find(&users)

	formatted := make([]map[string]interface{}, len(users))
	for i, u := range users {
		formatted[i] = formatUser(u)
	}
	return formatted, total, nil
}

// Get 单条用户
func (s *AdminUserService) Get(id int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrUserInvalidID
	}
	var user model.AdminUsers
	if err := model.DB.First(&user, id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return formatUser(user), nil
}

// Create 创建用户
func (s *AdminUserService) Create(username, password, email, phone string, status int, roleID uint) (map[string]interface{}, error) {
	var existing model.AdminUsers
	if err := model.DB.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, ErrUserNameDuplicate
	}

	user := model.AdminUsers{
		Username: username,
		Email:    email,
		Phone:    phone,
		Status:   status,
		RoleID:   roleID,
	}

	if err := user.SetPassword(password); err != nil {
		return nil, ErrUserPasswordHash
	}

	if err := model.DB.Create(&user).Error; err != nil {
		return nil, ErrUserCreate
	}

	return formatUser(user), nil
}

// Update 更新用户
func (s *AdminUserService) Update(id int, email, phone string, status int, roleID uint) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrUserInvalidID
	}
	var user model.AdminUsers
	if err := model.DB.First(&user, id).Error; err != nil {
		return nil, ErrUserNotFound
	}

	updates := map[string]interface{}{
		"email":   email,
		"phone":   phone,
		"status":  status,
		"role_id": roleID,
	}

	if err := model.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, ErrUserUpdate
	}

	model.DB.First(&user, id)
	return formatUser(user), nil
}

// Delete 删除用户(走 gorm 软删除)
func (s *AdminUserService) Delete(id int) error {
	if id <= 0 {
		return ErrUserInvalidID
	}
	if err := model.DB.Unscoped().Delete(&model.AdminUsers{}, id).Error; err != nil {
		return ErrUserDelete
	}
	return nil
}

// UpdateMe 个人中心 - 当前用户改自己(不依赖 PermissionMiddleware)
//
// 2026-10-19 新增:让"个人中心"弹窗能改邮箱/手机号/密码
//   - id: 从 c.Get("user_id") 拿,前端不传(防止越权)
//   - email/phone: 可选,空字符串 = 不改
//   - newPassword: 可选,空字符串 = 不改
//   - oldPassword: 必填(任何修改都要求输入当前密码二次确认)
//     即使只改邮箱/手机号,也要验证旧密码,防被人留在电脑前被改
//   - 拒绝改 username / role_id / status / department_id(只读)
func (s *AdminUserService) UpdateMe(id uint, email, phone, newPassword, oldPassword string) (map[string]interface{}, error) {
	if id == 0 {
		return nil, ErrUserInvalidID
	}
	if oldPassword == "" {
		return nil, ErrUserPasswordWrong
	}

	var user model.AdminUsers
	if err := model.DB.First(&user, id).Error; err != nil {
		return nil, ErrUserNotFound
	}

	// 验证旧密码(防越权 — 任何修改都要先确认当前密码)
	if !user.CheckPassword(oldPassword) {
		return nil, ErrUserPasswordWrong
	}

	updates := map[string]interface{}{}
	if email != "" {
		updates["email"] = email
	}
	if phone != "" {
		updates["phone"] = phone
	}
	if newPassword != "" {
		// 改密码:用 SetPassword 走 bcrypt
		if err := user.SetPassword(newPassword); err != nil {
			return nil, ErrUserPasswordHash
		}
		updates["password"] = user.Password
	}

	// 什么字段都没传(只验证了旧密码),不写库
	if len(updates) == 0 {
		return formatUser(user), nil
	}

	if err := model.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, ErrUserUpdate
	}

	model.DB.First(&user, id)
	return formatUser(user), nil
}
