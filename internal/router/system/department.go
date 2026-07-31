// Package system department.go - 部门管理路由
//
// 路由组: /api/system/departments
package system

import (
	"go_server/internal/handler/system"
	"go_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// DepartmentRoutes 部门管理路由
func DepartmentRoutes(rg *gin.RouterGroup) {
	depts := rg.Group("/system/departments")
	depts.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
	{
		depts.GET("", system.GetDepartments)
	}
}
