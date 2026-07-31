// Package system department.go - 部门管理 HTTP 入口
//
// 给"数据范围权限"的"看部门"用
//
// 接口列表:
//  GET /api/system/departments  部门列表(扁平)
package system

import (
	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// deptSvc 部门管理业务入口
var deptSvc = service.NewDepartmentService()

// GetDepartments 部门列表
func GetDepartments(c *gin.Context) {
	list, err := deptSvc.GetList()
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "获取部门列表失败")
		return
	}
	handler.Success(c, list)
}
