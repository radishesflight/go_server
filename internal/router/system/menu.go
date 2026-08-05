// Package system menu.go - 菜单管理路由
//
// 替代旧的 adminMenus.go
// 路由组: /api/system/adminMenus(注意:路径段跟 menu.code 一致,中间件推断 adminMenus:*)
// 加了 /operations/:menu_id 用于查某菜单的所有 operation
package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// MenuRoutes 菜单管理路由
func MenuRoutes(rg *gin.RouterGroup) {
	menus := rg.Group("/system/adminMenus")
	menus.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		menus.GET("/list", system.GetMenusList)
		menus.GET("/all", system.GetAllMenus)
		menus.GET("/options", system.GetMenusOptions)
		menus.GET("/operations/:menu_id", system.GetMenuOperations)
		menus.GET("/:id", system.GetMenu)
		menus.POST("", system.CreateMenu)
		menus.PUT("/:id", system.UpdateMenu)
		menus.DELETE("/:id", system.DeleteMenu)

		// operation(admin_menu_operations)CRUD
		// 路径段 /operations/* 跟 :id 不冲突(因为 :id 走的是数字 ID,operations 是固定前缀)
		menus.GET("/operations/get/:id", system.GetOperation)
		menus.POST("/operations", system.CreateOperation)
		menus.PUT("/operations/:id", system.UpdateOperation)
		menus.DELETE("/operations/:id", system.DeleteOperation)
	}
}
