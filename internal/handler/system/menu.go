// Package system menu.go - 菜单管理 HTTP 入口
//
// 替代旧的 adminMenus.go
// 加了 operations 关联查询接口
//
// 接口列表:
//
//	GET    /api/system/adminMenus/list                    列表分页查询
//	GET    /api/system/adminMenus/all                     所有菜单(带 operations)
//	GET    /api/system/adminMenus/options                 父级菜单下拉选项
//	GET    /api/system/adminMenus/operations/:menu_id     某菜单的所有 operation
//	GET    /api/system/adminMenus/operations/get/:id      operation 单条
//	POST   /api/system/adminMenus/operations              operation 新增
//	PUT    /api/system/adminMenus/operations/:id          operation 更新
//	DELETE /api/system/adminMenus/operations/:id          operation 删除
//	GET    /api/system/adminMenus/:id                     单条
//	POST   /api/system/adminMenus                         新增
//	PUT    /api/system/adminMenus/:id                     更新
//	DELETE /api/system/adminMenus/:id                     删除
//
// 业务码翻译表:
//
//	service.ErrMenuNotFound      → CodeMenuNotFound (1010)
//	service.ErrOperationNotFound → CodeMenuNotFound (1010,operation 复用)
//	service.ErrOperationDuplicate → CodeParamsInvalid (4001)
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

// =====================================================
// operation (admin_menu_operations) HTTP 入口
// =====================================================
//
// 操作 = 一条具体的 (method, path) 权限元数据
// 4 个 CRUD 接口,供"菜单管理 -> 操作"页面用

// CreateOperationReq 创建 operation 请求体
type CreateOperationReq struct {
	MenuID uint   `json:"menu_id" binding:"required"`
	Method string `json:"method" binding:"required"`
	Path   string `json:"path" binding:"required"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
}

// UpdateOperationReq 更新 operation 请求体
type UpdateOperationReq struct {
	MenuID uint   `json:"menu_id"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
}

// GetOperation 单条 operation
// GET /api/system/adminMenus/operations/get/:id
func GetOperation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的操作ID")
		return
	}
	op, err := menuSvc.GetOperation(id)
	if err != nil {
		if errors.Is(err, service.ErrOperationNotFound) {
			handler.Error(c, handler.CodeMenuNotFound, "操作不存在")
		} else {
			handler.Error(c, handler.CodeParamsInvalid, "无效的操作ID")
		}
		return
	}
	handler.Success(c, op)
}

// CreateOperation 新增 operation
// POST /api/system/adminMenus/operations
func CreateOperation(c *gin.Context) {
	var req CreateOperationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	op, err := menuSvc.CreateOperation(req.MenuID, req.Method, req.Path, req.Name, req.Sort)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOperationDuplicate):
			handler.Error(c, handler.CodeParamsInvalid, "该菜单下已存在相同的 (method, path) 操作")
		case errors.Is(err, service.ErrMenuNotFound):
			handler.Error(c, handler.CodeMenuNotFound, "菜单不存在")
		case errors.Is(err, service.ErrMenuInvalidID), errors.Is(err, service.ErrOperationInvalid):
			handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		default:
			handler.Error(c, handler.CodeUnknown, "创建操作失败")
		}
		return
	}
	handler.Success(c, op)
}

// UpdateOperation 更新 operation
// PUT /api/system/adminMenus/operations/:id
func UpdateOperation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的操作ID")
		return
	}

	var req UpdateOperationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	op, err := menuSvc.UpdateOperation(id, req.MenuID, req.Name, req.Sort)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOperationNotFound):
			handler.Error(c, handler.CodeMenuNotFound, "操作不存在")
		case errors.Is(err, service.ErrOperationInvalid):
			handler.Error(c, handler.CodeParamsInvalid, "无效的操作ID")
		default:
			handler.Error(c, handler.CodeUnknown, "更新操作失败")
		}
		return
	}
	handler.Success(c, op)
}

// DeleteOperation 删除 operation
// DELETE /api/system/adminMenus/operations/:id
func DeleteOperation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的操作ID")
		return
	}

	if err := menuSvc.DeleteOperation(id); err != nil {
		switch {
		case errors.Is(err, service.ErrOperationNotFound):
			handler.Error(c, handler.CodeMenuNotFound, "操作不存在")
		case errors.Is(err, service.ErrOperationInvalid):
			handler.Error(c, handler.CodeParamsInvalid, "无效的操作ID")
		default:
			handler.Error(c, handler.CodeUnknown, "删除操作失败")
		}
		return
	}
	handler.Success(c, nil)
}
