// Package system menu.go - 菜单管理 HTTP 入口
//
// 替代旧的 adminMenus.go
// 加了 operations 关联查询接口
//
// 接口列表:
//  GET    /api/system/menus/list              列表分页查询
//  GET    /api/system/menus/all               所有菜单(带 operations)
//  GET    /api/system/menus/options           父级菜单下拉选项
//  GET    /api/system/menus/operations/:menu_id  某菜单的所有操作
//  GET    /api/system/menus/:id               单条
//  POST   /api/system/menus                   新增
//  PUT    /api/system/menus/:id               更新
//  DELETE /api/system/menus/:id               删除
//
// 业务码翻译表:
//  service.ErrMenuNotFound → CodeMenuNotFound (1010)
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// menuSvc 菜单管理业务入口
var menuSvc = service.NewMenuService()

// MenusQuery 列表查询参数
type MenusQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

// CreateMenuReq 创建请求体
type CreateMenuReq struct {
	Name      string          `json:"name" binding:"required"`
	Code      string          `json:"code" binding:"required"`
	Path      string          `json:"path"`
	Icon      string          `json:"icon"`
	ParentID  uint            `json:"parent_id"`
	Sort      service.SortInt `json:"sort"`
	Status    int             `json:"status"`
	DataScope int             `json:"data_scope"`
}

// UpdateMenuReq 更新请求体
type UpdateMenuReq struct {
	Name      string          `json:"name"`
	Code      string          `json:"code"`
	Path      string          `json:"path"`
	Icon      string          `json:"icon"`
	ParentID  uint            `json:"parent_id"`
	Sort      service.SortInt `json:"sort"`
	Status    int             `json:"status"`
	DataScope int             `json:"data_scope"`
}

// GetMenusList 分页查询菜单列表
func GetMenusList(c *gin.Context) {
	var query MenusQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	list, total, _ := menuSvc.GetList(query.Page, query.Size, query.Status)
	handler.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  query.Page,
		"size":  query.Size,
	})
}

// GetAllMenus 所有菜单(带 operations)
func GetAllMenus(c *gin.Context) {
	list := menuSvc.GetAllWithOperations()
	handler.Success(c, list)
}

// GetMenusOptions 父级菜单下拉选项
func GetMenusOptions(c *gin.Context) {
	options := menuSvc.GetOptions()
	handler.Success(c, options)
}

// GetMenuOperations 某菜单的所有 operation
func GetMenuOperations(c *gin.Context) {
	menuID, err := strconv.Atoi(c.Param("menu_id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}
	ops := menuSvc.GetOperationsByMenuID(uint(menuID))
	handler.Success(c, ops)
}

// GetMenu 单条菜单
func GetMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}

	m, err := menuSvc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrMenuNotFound) {
			handler.Error(c, handler.CodeMenuNotFound, "菜单不存在")
		} else {
			handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		}
		return
	}
	handler.Success(c, m)
}

// CreateMenu 创建菜单
func CreateMenu(c *gin.Context) {
	var req CreateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	m, err := menuSvc.Create(req.Name, req.Code, req.Path, req.Icon, req.ParentID, int(req.Sort), req.Status, req.DataScope)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "创建菜单失败")
		return
	}
	handler.Success(c, m)
}

// UpdateMenu 更新菜单
func UpdateMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}

	var req UpdateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, err.Error())
		return
	}

	m, err := menuSvc.Update(id, req.Name, req.Code, req.Path, req.Icon, req.ParentID, int(req.Sort), req.Status, req.DataScope)
	if err != nil {
		if errors.Is(err, service.ErrMenuNotFound) {
			handler.Error(c, handler.CodeMenuNotFound, "菜单不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "更新菜单失败")
		}
		return
	}
	handler.Success(c, m)
}

// DeleteMenu 删除菜单
func DeleteMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}

	if err := menuSvc.Delete(id); err != nil {
		handler.Error(c, handler.CodeUnknown, "删除菜单失败")
		return
	}
	handler.Success(c, nil)
}
