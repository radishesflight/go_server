// Package service endpoint_service.go - 部署端字典
//
// 端是固定 4 个(苹果/安卓/前台web/后台web),走 code_endpoints 表便于后续扩展
// 当前只提供"查所有端"接口,不做 CRUD(端字典由 seed 初始化)
package service

import (
	"errors"

	"go_server/internal/model"
)

// 业务错误
var (
	ErrEndpointNotFound = errors.New("端不存在")
)

// EndpointService 端字典业务
type EndpointService struct{}

// NewEndpointService 构造 EndpointService
func NewEndpointService() *EndpointService { return &EndpointService{} }

// formatEndpoint 把 model 序列化成给前端的 map
func formatEndpoint(ep model.CodeEndpoints) map[string]interface{} {
	return map[string]interface{}{
		"id":         ep.ID,
		"code":       ep.Code,
		"name":       ep.Name,
		"ext":        ep.Ext,
		"icon":       ep.Icon,
		"sort":       ep.Sort,
		"status":     ep.Status,
		"created_at": model.FormatTime(ep.CreatedAt),
		"updated_at": model.FormatTime(ep.UpdatedAt),
	}
}

// ListAll 查询所有启用的端(按 sort asc)
func (s *EndpointService) ListAll() ([]map[string]interface{}, error) {
	var eps []model.CodeEndpoints
	if err := model.DB.Where("status = ?", 1).Order("sort asc").Find(&eps).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(eps))
	for i, e := range eps {
		out[i] = formatEndpoint(e)
	}
	return out, nil
}

// GetByID 单条
func (s *EndpointService) GetByID(id uint) (model.CodeEndpoints, error) {
	var ep model.CodeEndpoints
	if err := model.DB.First(&ep, id).Error; err != nil {
		return ep, ErrEndpointNotFound
	}
	return ep, nil
}

// GetByCode 机器名查
func (s *EndpointService) GetByCode(code string) (model.CodeEndpoints, error) {
	var ep model.CodeEndpoints
	if err := model.DB.Where("code = ?", code).First(&ep).Error; err != nil {
		return ep, ErrEndpointNotFound
	}
	return ep, nil
}
