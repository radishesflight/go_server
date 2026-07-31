// Package service menu_service.go - 菜单管理业务
//
// 替代旧的 admin_menu_service.go
// 加了 operations 关联(每个菜单支持的操作)和 dataScope 字段
//
// SortInt 兼容 string 和 int 的 JSON 解析
//   "10" → 10 / 10 → 10 / "" → 0 / "abc" → 0
package service

import (
	"errors"
	"strconv"

	"go_server/internal/model"
)

var (
	ErrMenuInvalidID = errors.New("无效的菜单ID")
	ErrMenuNotFound  = errors.New("菜单不存在")
	ErrMenuCreate    = errors.New("创建菜单失败")
	ErrMenuUpdate    = errors.New("更新菜单失败")
	ErrMenuDelete    = errors.New("删除菜单失败")
)

// SortInt 兼容 string 和 int 的 JSON 解析
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

// MenuService 菜单管理业务
type MenuService struct{}

// NewMenuService 构造 MenuService
func NewMenuService() *MenuService { return &MenuService{} }

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
		"data_scope": menu.DataScope,
		"created_at": model.FormatTime(menu.CreatedAt),
		"updated_at": model.FormatTime(menu.UpdatedAt),
	}
}

func formatOperation(op model.AdminMenuOperations) map[string]interface{} {
	return map[string]interface{}{
		"id":         op.ID,
		"menu_id":    op.MenuID,
		"code":       op.Code,
		"name":       op.Name,
		"icon":       op.Icon,
		"sort":       op.Sort,
		"created_at": model.FormatTime(op.CreatedAt),
		"updated_at": model.FormatTime(op.UpdatedAt),
	}
}

// GetList 分页查询菜单列表
func (s *MenuService) GetList(page, size, status int) ([]map[string]interface{}, int64, error) {
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
func (s *MenuService) Get(id int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrMenuInvalidID
	}
	var menu model.AdminMenus
	if err := model.DB.First(&menu, id).Error; err != nil {
		return nil, ErrMenuNotFound
	}
	return formatMenu(menu), nil
}

// GetAllWithOperations 返回所有菜单(带 operations)
// 给"分配菜单"页面用
func (s *MenuService) GetAllWithOperations() []map[string]interface{} {
	var menus []model.AdminMenus
	model.DB.Order("sort DESC").Find(&menus)

	menuIDs := make([]uint, len(menus))
	for i, m := range menus {
		menuIDs[i] = m.ID
	}

	// 一次性查所有 operations
	var ops []model.AdminMenuOperations
	if len(menuIDs) > 0 {
		model.DB.Where("menu_id IN ?", menuIDs).Order("sort asc").Find(&ops)
	}

	// 按 menu_id 分组
	opsMap := make(map[uint][]model.AdminMenuOperations)
	for _, op := range ops {
		opsMap[op.MenuID] = append(opsMap[op.MenuID], op)
	}

	formatted := make([]map[string]interface{}, len(menus))
	for i, m := range menus {
		item := formatMenu(m)
		item["operations"] = formatOperationsList(opsMap[m.ID])
		formatted[i] = item
	}
	return formatted
}

// GetOperationsByMenuID 查某菜单的所有 operation
func (s *MenuService) GetOperationsByMenuID(menuID uint) []map[string]interface{} {
	var ops []model.AdminMenuOperations
	model.DB.Where("menu_id = ?", menuID).Order("sort asc").Find(&ops)
	return formatOperationsList(ops)
}

func formatOperationsList(ops []model.AdminMenuOperations) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(ops))
	for i, op := range ops {
		formatted[i] = formatOperation(op)
	}
	return formatted
}

// GetOptions parent_id=0 的菜单
func (s *MenuService) GetOptions() []map[string]interface{} {
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
func (s *MenuService) Create(name, code, path, icon string, parentID uint, sort, status, dataScope int) (map[string]interface{}, error) {
	menu := model.AdminMenus{
		Name:      name,
		Code:      code,
		Path:      path,
		Icon:      icon,
		ParentID:  parentID,
		Sort:      sort,
		Status:    status,
		DataScope: dataScope,
	}
	if err := model.DB.Create(&menu).Error; err != nil {
		return nil, ErrMenuCreate
	}
	return formatMenu(menu), nil
}

// Update 更新菜单
func (s *MenuService) Update(id int, name, code, path, icon string, parentID uint, sort, status, dataScope int) (map[string]interface{}, error) {
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
	updates["data_scope"] = dataScope

	if err := model.DB.Model(&menu).Updates(updates).Error; err != nil {
		return nil, ErrMenuUpdate
	}
	model.DB.First(&menu, id)
	return formatMenu(menu), nil
}

// Delete 删除菜单
func (s *MenuService) Delete(id int) error {
	if id <= 0 {
		return ErrMenuInvalidID
	}
	return model.DB.Delete(&model.AdminMenus{}, id).Error
}
