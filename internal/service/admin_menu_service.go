// Package service admin_menu_service.go - 菜单管理业务
//
// CRUD + SortInt(兼容前端传 string "10" 或 int 10)
// 业务错误:
//  ErrMenuInvalidID → 无效的 ID
//  ErrMenuNotFound  → 菜单不存在
//
// SortInt 设计:前端可能用 el-input number 传 string,UnmarshalJSON 兼容
//   "10" → 10
//   10   → 10
//   ""   → 0
//   "abc" → 0
package service

import (
	"errors"
	"strconv"

	"go_server/internal/model"
)

// 业务错误(与原 handler 中的错误文案对应)
var (
	ErrMenuInvalidID = errors.New("无效的菜单ID")
	ErrMenuNotFound  = errors.New("菜单不存在")
	ErrMenuCreate    = errors.New("创建菜单失败")
	ErrMenuUpdate    = errors.New("更新菜单失败")
	ErrMenuDelete    = errors.New("删除菜单失败")
)

// SortInt 兼容 string 和 int 的 JSON 解析
// (从原 handler/system/adminMenus.go 平移过来)
type SortInt int

func (s *SortInt) UnmarshalJSON(data []byte) error {
	str := string(data)
	str = trimQuotes(str)
	if str == "" {
		*s = 0
		return nil
	}
	if n, err := strconv.Atoi(str); err == nil {
		*s = SortInt(n)
		return nil
	}
	*s = 0
	return nil
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// AdminMenuService 菜单管理业务
type AdminMenuService struct{}

// NewAdminMenuService 构造 AdminMenuService
func NewAdminMenuService() *AdminMenuService { return &AdminMenuService{} }

// formatMenu 把菜单 model 序列化成给前端的 map
func formatMenu(menu model.AdminMenus) map[string]interface{} {
	return map[string]interface{}{
		"id":         menu.ID,
		"name":       menu.Name,
		"code":       menu.Code,
		"path":       menu.Path,
		"icon":       menu.Icon,
		"parent_id":  menu.ParentID,
		"sort":       menu.Sort,
		"status":     menu.Status,
		"buttons":    menu.Buttons,
		"created_at": model.FormatTime(menu.CreatedAt),
		"updated_at": model.FormatTime(menu.UpdatedAt),
	}
}

// GetList 分页查询菜单列表
func (s *AdminMenuService) GetList(page, size, status int) ([]map[string]interface{}, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var menus []model.AdminMenus
	var total int64

	db := model.DB.Model(&model.AdminMenus{})
	if status > 0 {
		db = db.Where("status = ?", status)
	}

	db.Count(&total)
	db.Order("sort asc").Offset((page - 1) * size).Limit(size).Find(&menus)

	formatted := make([]map[string]interface{}, len(menus))
	for i, m := range menus {
		formatted[i] = formatMenu(m)
	}
	return formatted, total, nil
}

// Get 单条菜单
func (s *AdminMenuService) Get(id int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrMenuInvalidID
	}
	var menu model.AdminMenus
	if err := model.DB.First(&menu, id).Error; err != nil {
		return nil, ErrMenuNotFound
	}
	return formatMenu(menu), nil
}

// GetAll 所有菜单(sort DESC,与原 GetAllMenus 一致)
func (s *AdminMenuService) GetAll() []map[string]interface{} {
	var menus []model.AdminMenus
	model.DB.Order("sort DESC").Find(&menus)
	formatted := make([]map[string]interface{}, len(menus))
	for i, m := range menus {
		formatted[i] = formatMenu(m)
	}
	return formatted
}

// GetOptions parent_id=0 的菜单(id + name),给前端下拉用
func (s *AdminMenuService) GetOptions() []map[string]interface{} {
	var menus []model.AdminMenus
	model.DB.Where("parent_id = ?", 0).Order("sort asc").Find(&menus)
	options := make([]map[string]interface{}, len(menus))
	for i, m := range menus {
		options[i] = map[string]interface{}{
			"id":   m.ID,
			"name": m.Name,
		}
	}
	return options
}

// Create 创建菜单
func (s *AdminMenuService) Create(name, code, path, icon, buttons string, parentID uint, sort int, status int) (map[string]interface{}, error) {
	menu := model.AdminMenus{
		Name:     name,
		Code:     code,
		Path:     path,
		Icon:     icon,
		ParentID: parentID,
		Sort:     sort,
		Status:   status,
		Buttons:  buttons,
	}
	if err := model.DB.Create(&menu).Error; err != nil {
		return nil, ErrMenuCreate
	}
	return formatMenu(menu), nil
}

// Update 更新菜单
// 注意:原 handler 对 Update 的更新字段是无条件覆盖(空字符串也会写入),保持一致
func (s *AdminMenuService) Update(id int, name, code, path, icon, buttons string, parentID uint, sort int, status int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrMenuInvalidID
	}
	var menu model.AdminMenus
	if err := model.DB.First(&menu, id).Error; err != nil {
		return nil, ErrMenuNotFound
	}

	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if code != "" {
		updates["code"] = code
	}
	updates["path"] = path
	updates["icon"] = icon
	updates["parent_id"] = parentID
	updates["sort"] = sort
	updates["status"] = status
	updates["buttons"] = buttons

	if err := model.DB.Model(&menu).Updates(updates).Error; err != nil {
		return nil, ErrMenuUpdate
	}
	model.DB.First(&menu, id)
	return formatMenu(menu), nil
}

// Delete 删除菜单
func (s *AdminMenuService) Delete(id int) error {
	if id <= 0 {
		return ErrMenuInvalidID
	}
	if err := model.DB.Delete(&model.AdminMenus{}, id).Error; err != nil {
		return ErrMenuDelete
	}
	return nil
}
