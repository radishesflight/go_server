package service

import (
	"errors"

	"go_server/internal/model"
)

// 业务错误(与原 handler 中的错误文案对应)
var (
	ErrRoleInvalidID     = errors.New("无效的角色ID")
	ErrRoleNotFound      = errors.New("角色不存在")
	ErrRoleNameDuplicate = errors.New("角色名称已存在")
	ErrRoleCreate        = errors.New("创建角色失败")
	ErrRoleUpdate        = errors.New("更新角色失败")
	ErrRoleDelete        = errors.New("删除角色失败")
)

// AdminRoleService 角色管理业务
type AdminRoleService struct{}

// NewAdminRoleService 构造 AdminRoleService
func NewAdminRoleService() *AdminRoleService { return &AdminRoleService{} }

// formatRole 把角色 model 序列化成给前端的 map
func formatRole(role model.AdminRoles) map[string]interface{} {
	return map[string]interface{}{
		"id":         role.ID,
		"name":       role.Name,
		"describe":   role.Describe,
		"status":     role.Status,
		"created_at": model.FormatTime(role.CreatedAt),
		"updated_at": model.FormatTime(role.UpdatedAt),
	}
}

// GetList 分页查询角色列表
func (s *AdminRoleService) GetList(page, size, status int) ([]map[string]interface{}, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var roles []model.AdminRoles
	var total int64

	db := model.DB.Model(&model.AdminRoles{})
	if status > 0 {
		db = db.Where("status = ?", status)
	}

	db.Count(&total)
	db.Offset((page - 1) * size).Limit(size).Find(&roles)

	formatted := make([]map[string]interface{}, len(roles))
	for i, r := range roles {
		formatted[i] = formatRole(r)
	}
	return formatted, total, nil
}

// Get 单条角色
func (s *AdminRoleService) Get(id int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrRoleInvalidID
	}
	var role model.AdminRoles
	if err := model.DB.First(&role, id).Error; err != nil {
		return nil, ErrRoleNotFound
	}
	return formatRole(role), nil
}

// Create 创建角色
func (s *AdminRoleService) Create(name, describe string, status int) (map[string]interface{}, error) {
	var existing model.AdminRoles
	if err := model.DB.Where("name = ?", name).First(&existing).Error; err == nil {
		return nil, ErrRoleNameDuplicate
	}

	role := model.AdminRoles{
		Name:     name,
		Describe: describe,
		Status:   status,
	}
	if err := model.DB.Create(&role).Error; err != nil {
		return nil, ErrRoleCreate
	}
	return formatRole(role), nil
}

// Update 更新角色
func (s *AdminRoleService) Update(id int, name, describe string, status int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrRoleInvalidID
	}
	var role model.AdminRoles
	if err := model.DB.First(&role, id).Error; err != nil {
		return nil, ErrRoleNotFound
	}

	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	updates["describe"] = describe
	updates["status"] = status

	if err := model.DB.Model(&role).Updates(updates).Error; err != nil {
		return nil, ErrRoleUpdate
	}
	model.DB.First(&role, id)
	return formatRole(role), nil
}

// Delete 删除角色
func (s *AdminRoleService) Delete(id int) error {
	if id <= 0 {
		return ErrRoleInvalidID
	}
	if err := model.DB.Delete(&model.AdminRoles{}, id).Error; err != nil {
		return ErrRoleDelete
	}
	return nil
}
