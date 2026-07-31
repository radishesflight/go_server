// Package system adminMenus.go - 菜单管理 HTTP 入口
//
// 接口列表:
//  GET    /api/system/adminMenus/list      列表分页查询(sort asc)
//  GET    /api/system/adminMenus/all       所有菜单(sort DESC,用于角色分配)
//  GET    /api/system/adminMenus/options   parent_id=0 的菜单(用于"上级菜单"下拉)
//  GET    /api/system/adminMenus/:id       单条
//  POST   /api/system/adminMenus           新增
//  PUT    /api/system/adminMenus/:id       更新
//  DELETE /api/system/adminMenus/:id       删除
//
// 注意:Update 接口是"按字段覆盖"模式(空字符串/0 也会写入 DB),
// 与 adminRoles.Update 的"name 不更新"不一致,是有意为之。
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

// adminMenuSvc 菜单管理业务入口
var adminMenuSvc = service.NewAdminMenuService()

// AdminMenusQuery 列表查询参数
type AdminMenusQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

// CreateAdminMenusReq 创建请求体
// Sort 用 service.SortInt(兼容前端传 string "10" 或 int 10)
type CreateAdminMenusReq struct {
	Name     string          `json:"name" binding:"required"`  // 菜单名
	Code     string          `json:"code" binding:"required"`  // 编码(用于权限推断,如 "adminUsers")
	Path     string          `json:"path"`                     // 路由路径
	Icon     string          `json:"icon"`                     // Element Plus 图标名
	ParentID uint            `json:"parent_id"`                // 0=顶级菜单
	Sort     service.SortInt `json:"sort"`                     // 排序
	Status   int             `json:"status"`                   // 1=启用,0=禁用
	Buttons  string          `json:"buttons"`                  // 预留字段
}

// UpdateAdminMenusReq 更新请求体
type UpdateAdminMenusReq struct {
	Name     string          `json:"name"`
	Code     string          `json:"code"`
	Path     string          `json:"path"`
	Icon     string          `json:"icon"`
	ParentID uint            `json:"parent_id"`
	Sort     service.SortInt `json:"sort"`
	Status   int             `json:"status"`
	Buttons  string          `json:"buttons"`
}

// GetAdminMenusList 分页查询菜单列表
func GetAdminMenusList(c *gin.Context) {
	var query AdminMenusQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	list, total, _ := adminMenuSvc.GetList(query.Page, query.Size, query.Status)

	handler.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  query.Page,
		"size":  query.Size,
	})
}

// GetAdminMenus 单条菜单
func GetAdminMenus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}

	m, err := adminMenuSvc.Get(id)
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

// GetAllMenus 所有菜单
func GetAllMenus(c *gin.Context) {
	list := adminMenuSvc.GetAll()
	handler.Success(c, list)
}

// GetAdminMenusOptions 父级菜单选项
func GetAdminMenusOptions(c *gin.Context) {
	options := adminMenuSvc.GetOptions()
	handler.Success(c, options)
}

// CreateAdminMenus 创建菜单
func CreateAdminMenus(c *gin.Context) {
	var req CreateAdminMenusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	m, err := adminMenuSvc.Create(req.Name, req.Code, req.Path, req.Icon, req.Buttons, req.ParentID, int(req.Sort), req.Status)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "创建菜单失败")
		return
	}
	handler.Success(c, m)
}

// UpdateAdminMenus 更新菜单
// 注意:与原 handler 行为一致,Update 走的是"按字段覆盖"模式(空字符串/0 也会写入)
func UpdateAdminMenus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}

	var req UpdateAdminMenusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, err.Error())
		return
	}

	m, err := adminMenuSvc.Update(id, req.Name, req.Code, req.Path, req.Icon, req.Buttons, req.ParentID, int(req.Sort), req.Status)
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

// DeleteAdminMenus 删除菜单
func DeleteAdminMenus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的菜单ID")
		return
	}

	if err := adminMenuSvc.Delete(id); err != nil {
		handler.Error(c, handler.CodeUnknown, "删除菜单失败")
		return
	}
	handler.Success(c, nil)
}
