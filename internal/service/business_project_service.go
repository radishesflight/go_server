// Package service business_project_service.go - 业务项目管理
//
// 业务项目 = 树父节点(如 b2b电商项目)
//   - 走 business_projects 表
//   - 通过 business_project_endpoints 关联端字典
//
// 树查询:ListTree 返回嵌套结构 [{id, name, endpoints: [{id, name, ext, icon}]}]
package service

import (
	"errors"

	"go_server/internal/model"
)

// 业务错误
var (
	ErrProjectNotFound   = errors.New("业务项目不存在")
	ErrProjectCodeExists = errors.New("业务项目编码已存在")
	ErrProjectCreate     = errors.New("创建业务项目失败")
	ErrProjectUpdate     = errors.New("更新业务项目失败")
	ErrProjectDelete     = errors.New("删除业务项目失败")
)

// BusinessProjectService 业务项目管理
type BusinessProjectService struct{}

// NewBusinessProjectService 构造
func NewBusinessProjectService() *BusinessProjectService { return &BusinessProjectService{} }

// formatProject 把 model 序列化成给前端的 map
func formatProject(p model.BusinessProjects) map[string]interface{} {
	return map[string]interface{}{
		"id":          p.ID,
		"code":        p.Code,
		"name":        p.Name,
		"description": p.Description,
		"sort":        p.Sort,
		"status":      p.Status,
		"created_at":  model.FormatTime(p.CreatedAt),
		"updated_at":  model.FormatTime(p.UpdatedAt),
	}
}

// List 查询所有启用的项目(按 sort asc)
func (s *BusinessProjectService) List() ([]map[string]interface{}, error) {
	var projects []model.BusinessProjects
	if err := model.DB.Where("status = ?", 1).Order("sort asc, id asc").Find(&projects).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		out[i] = formatProject(p)
	}
	return out, nil
}

// Get 单条
func (s *BusinessProjectService) Get(id uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, ErrProjectNotFound
	}
	var p model.BusinessProjects
	if err := model.DB.First(&p, id).Error; err != nil {
		return nil, ErrProjectNotFound
	}
	return formatProject(p), nil
}

// GetWithEndpoints 单条 + 该项目下的端列表
func (s *BusinessProjectService) GetWithEndpoints(id uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, ErrProjectNotFound
	}
	var p model.BusinessProjects
	if err := model.DB.First(&p, id).Error; err != nil {
		return nil, ErrProjectNotFound
	}
	eps, err := s.endpointsOf(p.ID)
	if err != nil {
		return nil, err
	}
	out := formatProject(p)
	out["endpoints"] = eps
	return out, nil
}

// Create 新建业务项目(同时设置该项目的端列表)
func (s *BusinessProjectService) Create(code, name, description string, sort, status int, endpointIDs []uint) (map[string]interface{}, error) {
	// code 唯一性
	var existing model.BusinessProjects
	if err := model.DB.Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, ErrProjectCodeExists
	}

	proj := model.BusinessProjects{
		Code:        code,
		Name:        name,
		Description: description,
		Sort:        sort,
		Status:      status,
	}
	if err := model.DB.Create(&proj).Error; err != nil {
		return nil, ErrProjectCreate
	}

	// 写端关联
	if err := s.setEndpoints(proj.ID, endpointIDs); err != nil {
		return nil, err
	}

	return s.GetWithEndpoints(proj.ID)
}

// Update 更新业务项目(全量替换端列表)
func (s *BusinessProjectService) Update(id uint, code, name, description string, sort, status int, endpointIDs []uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, ErrProjectNotFound
	}
	var proj model.BusinessProjects
	if err := model.DB.First(&proj, id).Error; err != nil {
		return nil, ErrProjectNotFound
	}

	// code 改了,要校验唯一
	if code != proj.Code {
		var dup model.BusinessProjects
		if err := model.DB.Where("code = ? AND id <> ?", code, id).First(&dup).Error; err == nil {
			return nil, ErrProjectCodeExists
		}
	}

	updates := map[string]interface{}{
		"code":        code,
		"name":        name,
		"description": description,
		"sort":        sort,
		"status":      status,
	}
	if err := model.DB.Model(&proj).Updates(updates).Error; err != nil {
		return nil, ErrProjectUpdate
	}

	if err := s.setEndpoints(id, endpointIDs); err != nil {
		return nil, err
	}
	return s.GetWithEndpoints(id)
}

// Delete 删除(软删除)
func (s *BusinessProjectService) Delete(id uint) error {
	if id == 0 {
		return ErrProjectNotFound
	}
	// 事务:删项目 + 删端关联
	tx := model.DB.Begin()
	if err := tx.Delete(&model.BusinessProjects{}, id).Error; err != nil {
		tx.Rollback()
		return ErrProjectDelete
	}
	if err := tx.Where("project_id = ?", id).Delete(&model.BusinessProjectEndpoints{}).Error; err != nil {
		tx.Rollback()
		return ErrProjectDelete
	}
	return tx.Commit().Error
}

// ListTree 返回整棵业务项目树
// 结构:[{id, code, name, description, sort, status, endpoints: [{id, code, name, ext, icon, sort}]}]
func (s *BusinessProjectService) ListTree() ([]map[string]interface{}, error) {
	projects, err := s.List()
	if err != nil {
		return nil, err
	}
	// 一次性查所有项目-端关联
	var rels []model.BusinessProjectEndpoints
	if err := model.DB.Where("status = ?", 1).Order("sort asc").Find(&rels).Error; err != nil {
		return nil, err
	}
	// 一次性查所有端
	var eps []model.CodeEndpoints
	if err := model.DB.Where("status = ?", 1).Order("sort asc").Find(&eps).Error; err != nil {
		return nil, err
	}
	epMap := make(map[uint]model.CodeEndpoints, len(eps))
	for _, e := range eps {
		epMap[e.ID] = e
	}
	// 按 project_id 聚合
	relByProject := make(map[uint][]uint)
	for _, r := range rels {
		relByProject[r.ProjectID] = append(relByProject[r.ProjectID], r.EndpointID)
	}

	for i, p := range projects {
		epIDs := relByProject[uint(p["id"].(uint))]
		epList := make([]map[string]interface{}, 0, len(epIDs))
		for _, eid := range epIDs {
			if e, ok := epMap[eid]; ok {
				epList = append(epList, map[string]interface{}{
					"id":   e.ID,
					"code": e.Code,
					"name": e.Name,
					"ext":  e.Ext,
					"icon": e.Icon,
					"sort": e.Sort,
				})
			}
		}
		projects[i]["endpoints"] = epList
	}
	return projects, nil
}

// endpointsOf 查询某项目下的端
func (s *BusinessProjectService) endpointsOf(projectID uint) ([]map[string]interface{}, error) {
	var rels []model.BusinessProjectEndpoints
	if err := model.DB.Where("project_id = ? AND status = ?", projectID, 1).Order("sort asc").Find(&rels).Error; err != nil {
		return nil, err
	}
	if len(rels) == 0 {
		return []map[string]interface{}{}, nil
	}
	epIDs := make([]uint, 0, len(rels))
	for _, r := range rels {
		epIDs = append(epIDs, r.EndpointID)
	}
	var eps []model.CodeEndpoints
	if err := model.DB.Where("id IN ?", epIDs).Order("sort asc").Find(&eps).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(eps))
	for i, e := range eps {
		out[i] = map[string]interface{}{
			"id":   e.ID,
			"code": e.Code,
			"name": e.Name,
			"ext":  e.Ext,
			"icon": e.Icon,
		}
	}
	return out, nil
}

// setEndpoints 全量替换某项目下的端关联
func (s *BusinessProjectService) setEndpoints(projectID uint, endpointIDs []uint) error {
	tx := model.DB.Begin()
	// 软删旧的
	if err := tx.Where("project_id = ?", projectID).Delete(&model.BusinessProjectEndpoints{}).Error; err != nil {
		tx.Rollback()
		return ErrProjectUpdate
	}
	// 插新的
	if len(endpointIDs) > 0 {
		rels := make([]model.BusinessProjectEndpoints, 0, len(endpointIDs))
		for i, eid := range endpointIDs {
			rels = append(rels, model.BusinessProjectEndpoints{
				ProjectID:  projectID,
				EndpointID: eid,
				Sort:       i,
				Status:     1,
			})
		}
		if err := tx.Create(&rels).Error; err != nil {
			tx.Rollback()
			return ErrProjectUpdate
		}
	}
	return tx.Commit().Error
}
