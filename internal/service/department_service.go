// Package service department_service.go - 部门管理业务
//
// 给"数据范围权限"的"看部门"用
// 部门是树形(parent_id 自引用)
package service

import (
	"go_server/internal/model"
)

// DepartmentService 部门管理业务
type DepartmentService struct{}

// NewDepartmentService 构造 DepartmentService
func NewDepartmentService() *DepartmentService { return &DepartmentService{} }

// GetList 部门列表(扁平)
func (s *DepartmentService) GetList() ([]map[string]interface{}, error) {
	var depts []model.AdminDepartments
	if err := model.DB.Order("sort asc").Find(&depts).Error; err != nil {
		return nil, err
	}
	formatted := make([]map[string]interface{}, len(depts))
	for i, d := range depts {
		formatted[i] = map[string]interface{}{
			"id":         d.ID,
			"name":       d.Name,
			"parent_id":  d.ParentID,
			"sort":       d.Sort,
			"status":     d.Status,
			"created_at": model.FormatTime(d.CreatedAt),
			"updated_at": model.FormatTime(d.UpdatedAt),
		}
	}
	return formatted, nil
}
