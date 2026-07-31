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

// 请求结构(JSON 字段不变,前端无感)
type AdminMenusQuery struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Status int `form:"status" json:"status"`
}

type CreateAdminMenusReq struct {
	Name     string          `json:"name" binding:"required"`
	Code     string          `json:"code" binding:"required"`
	Path     string          `json:"path"`
	Icon     string          `json:"icon"`
	ParentID uint            `json:"parent_id"`
	Sort     service.SortInt `json:"sort"`
	Status   int             `json:"status"`
	Buttons  string          `json:"buttons"`
}

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
