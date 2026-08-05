// Package service menu_service.go - 菜单管理业务
//
// 替代旧的 admin_menu_service.go
// 菜单关联 routes(admin_menu_operations 表),每条 route = (method, path)
//
// SortInt 兼容 string 和 int 的 JSON 解析
//
//	"10" → 10 / 10 → 10 / "" → 0 / "abc" → 0
package service

import (
	"errors"
	"strconv"
	"strings"

	"go_server/internal/model"
	"go_server/pkg/cache"
)

var (
	ErrMenuInvalidID      = errors.New("无效的菜单ID")
	ErrMenuNotFound       = errors.New("菜单不存在")
	ErrMenuCreate         = errors.New("创建菜单失败")
	ErrMenuUpdate         = errors.New("更新菜单失败")
	ErrMenuDelete         = errors.New("删除菜单失败")
	ErrOperationInvalid   = errors.New("无效的操作ID")
	ErrOperationNotFound  = errors.New("操作不存在")
	ErrOperationCreate    = errors.New("创建操作失败")
	ErrOperationUpdate    = errors.New("更新操作失败")
	ErrOperationDelete    = errors.New("删除操作失败")
	ErrOperationDuplicate = errors.New("该菜单下已存在相同的 (method, path) 操作")
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

// NewMenuService 构造
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

func formatRoute(r model.AdminMenuOperations) map[string]interface{} {
	return map[string]interface{}{
		"id":         r.ID,
		"menu_id":    r.MenuID,
		"method":     r.Method,
		"path":       r.Path,
		"name":       r.Name,
		"sort":       r.Sort,
		"created_at": model.FormatTime(r.CreatedAt),
		"updated_at": model.FormatTime(r.UpdatedAt),
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

// GetAllWithOperations 返回所有菜单(带 routes)
// 给"分配菜单"页面用 - 现在 routes 是 (method, path) 形式
func (s *MenuService) GetAllWithOperations() []map[string]interface{} {
	var menus []model.AdminMenus
	model.DB.Order("sort DESC").Find(&menus)

	menuIDs := make([]uint, len(menus))
	for i, m := range menus {
		menuIDs[i] = m.ID
	}

	// 一次性查所有 routes
	var routes []model.AdminMenuOperations
	if len(menuIDs) > 0 {
		model.DB.Where("menu_id IN ?", menuIDs).Order("sort asc").Find(&routes)
	}

	// 按 menu_id 分组
	routesMap := make(map[uint][]model.AdminMenuOperations)
	for _, r := range routes {
		routesMap[r.MenuID] = append(routesMap[r.MenuID], r)
	}

	formatted := make([]map[string]interface{}, len(menus))
	for i, m := range menus {
		item := formatMenu(m)
		item["operations"] = formatRoutesList(routesMap[m.ID]) // 字段名保留 "operations" 兼容前端
		formatted[i] = item
	}
	return formatted
}

// GetOperationsByMenuID 查某菜单的所有 route(兼容旧名字,内部用)
func (s *MenuService) GetOperationsByMenuID(menuID uint) []map[string]interface{} {
	var routes []model.AdminMenuOperations
	model.DB.Where("menu_id = ?", menuID).Order("sort asc").Find(&routes)
	return formatRoutesList(routes)
}

func formatRoutesList(routes []model.AdminMenuOperations) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(routes))
	for i, r := range routes {
		formatted[i] = formatRoute(r)
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

// =====================================================
// operation (admin_menu_operations) CRUD
// =====================================================
//
// 一条 operation = (method, path) 一条具体接口的权限元数据
// 由 admin_menu_operations 存储,role 通过 admin_role_operations 关联过来
//
// 加新接口的标准流程(现在可在前端页面完成,不用再 INSERT SQL):
//  1. 后端在 router/system/xxx.go 加 route
//  2. 重启服务,启动时 SyncRoutes 会自动把 (method, path) 同步到 admin_menu_operations
//  3. 管理员在"菜单管理 -> 操作"面板里修改中文名 / sort
//  4. 在"角色权限分配"里给需要的角色勾选

// GetOperation 单条 operation
func (s *MenuService) GetOperation(id int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrOperationInvalid
	}
	var op model.AdminMenuOperations
	if err := model.DB.First(&op, id).Error; err != nil {
		return nil, ErrOperationNotFound
	}
	return formatRoute(op), nil
}

// CreateOperation 新增 operation
// 同一 menu_id 下 (method, path) 唯一(uniqueIndex:idx_op_method_path)
func (s *MenuService) CreateOperation(menuID uint, method, path, name string, sort int) (map[string]interface{}, error) {
	if menuID == 0 {
		return nil, ErrMenuInvalidID
	}
	if method == "" || path == "" {
		return nil, ErrOperationInvalid
	}

	// 菜单存在性
	var menu model.AdminMenus
	if err := model.DB.First(&menu, menuID).Error; err != nil {
		return nil, ErrMenuNotFound
	}

	op := model.AdminMenuOperations{
		MenuID: menuID,
		Method: method,
		Path:   path,
		Name:   name,
		Sort:   sort,
	}
	if err := model.DB.Create(&op).Error; err != nil {
		// 唯一索引冲突
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "1062") {
			return nil, ErrOperationDuplicate
		}
		return nil, ErrOperationCreate
	}
	return formatRoute(op), nil
}

// UpdateOperation 更新 operation(只能改 name / sort / menu_id,不改 method + path)
func (s *MenuService) UpdateOperation(id int, menuID uint, name string, sort int) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, ErrOperationInvalid
	}
	var op model.AdminMenuOperations
	if err := model.DB.First(&op, id).Error; err != nil {
		return nil, ErrOperationNotFound
	}

	updates := map[string]interface{}{}
	if menuID > 0 && menuID != op.MenuID {
		updates["menu_id"] = menuID
	}
	if name != "" {
		updates["name"] = name
	}
	updates["sort"] = sort

	if len(updates) > 0 {
		if err := model.DB.Model(&op).Updates(updates).Error; err != nil {
			return nil, ErrOperationUpdate
		}
	}
	model.DB.First(&op, id)
	return formatRoute(op), nil
}

// DeleteOperation 删除 operation
// 同步删 admin_role_operations 里的关联(避免悬空 route_id)
// 然后 bump 所有关联角色的权限版本(让相关用户的 token 懒重载)
func (s *MenuService) DeleteOperation(id int) error {
	if id <= 0 {
		return ErrOperationInvalid
	}
	var op model.AdminMenuOperations
	if err := model.DB.First(&op, id).Error; err != nil {
		return ErrOperationNotFound
	}

	// 1. 找出哪些角色关联了这条 operation
	var roleOps []model.AdminRoleOperations
	model.DB.Where("route_id = ?", id).Find(&roleOps)
	roleIDs := make(map[uint]struct{}, len(roleOps))
	for _, ro := range roleOps {
		roleIDs[ro.RoleID] = struct{}{}
	}

	// 2. 删角色关联
	if err := model.DB.Where("route_id = ?", id).Delete(&model.AdminRoleOperations{}).Error; err != nil {
		return ErrOperationDelete
	}
	// 3. 删 operation
	if err := model.DB.Delete(&op).Error; err != nil {
		return ErrOperationDelete
	}
	// 4. bump 这些角色的权限版本 → 在线用户下次请求懒重载
	for rid := range roleIDs {
		go cache.BumpVersion(rid)
	}
	return nil
}
