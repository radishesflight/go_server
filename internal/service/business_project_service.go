// Package service business_project_service.go - 业务项目管理
//
// 业务项目 = 树父节点(如 b2b电商项目)
//   - 走 business_projects 表
//   - 通过 business_project_endpoints 关联端字典
//
// 树查询:ListTree 返回嵌套结构 [{id, name, endpoints: [{id, name, ext, icon}]}]
//
// 数据来源:
//   - CRUD(Create/Update/Delete):管理员手动维护
//   - SyncFromJMS:从 JumpServer Core API 拉节点树,增量 upsert
//     策略:按 code 匹配,JMS 新增 → INSERT,JMS 改名 → UPDATE name,JMS 消失 → 软删
package service

import (
	"errors"
	"regexp"

	"go_server/internal/model"
	"go_server/pkg/jumpserver"
)

// stripJMSCountSuffix 去掉 JMS 节点名末尾 " (数字)" 后缀(表示该节点下有多少资产)
//
// 例: "新丝路 (3)" → "新丝路"
//
//	   "一心堂 (4)"  → "一心堂"
//	- 末尾 " (数字)" 紧贴字符串结尾
//	- 数字可 0+ 位
var jmsCountSuffix = regexp.MustCompile(`\s*\(\d+\)\s*$`)

func stripJMSCountSuffix(name string) string {
	return jmsCountSuffix.ReplaceAllString(name, "")
}

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

// GetList 分页查询(给后台管理页用,包含已禁用的)
func (s *BusinessProjectService) GetList(page, size int) ([]map[string]interface{}, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	var projects []model.BusinessProjects
	var total int64
	db := model.DB.Model(&model.BusinessProjects{})
	db.Count(&total)
	db.Order("sort asc, id asc").Offset((page - 1) * size).Limit(size).Find(&projects)
	out := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		out[i] = formatProject(p)
	}
	return out, total, nil
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

// ListWithAssetsAndPaths 返回项目 + 资产 + 路径 三层嵌套(给"apk/前端包管理"页左侧树用)
//
// 结构:
//
//	[{
//	  id, code, name, ..., path_count,            // 项目的"端数量"= 路径总条数
//	  assets: [{
//	    id, name, address, ..., paths: [{id, name}] // 资产下的路径(只展示 name)
//	  }]
//	}]
//
// 一次性查 project + asset + path 三个表,然后内存组装,避免 N+1
//
// 注意:business_asset_paths 是真删(2026-10-16 起),所以 list 不用过滤 deleted_at
func (s *BusinessProjectService) ListWithAssetsAndPaths() ([]map[string]interface{}, error) {
	// 1. 查所有启用的项目
	var projects []model.BusinessProjects
	if err := model.DB.Where("status = ?", 1).Order("sort asc, id asc").Find(&projects).Error; err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return []map[string]interface{}{}, nil
	}

	projectIDs := make([]uint, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// 2. 查所有启用的资产(用 project_id IN 一次拿完)
	var assets []model.BusinessAssets
	if err := model.DB.Where("project_id IN ? AND status = ?", projectIDs, 1).
		Order("id asc").Find(&assets).Error; err != nil {
		return nil, err
	}

	// 3. 查所有路径(用 asset_id IN 一次拿完,business_asset_paths 真删,无 deleted_at 过滤)
	var paths []model.BusinessAssetPaths
	if len(assets) > 0 {
		assetIDs := make([]uint, len(assets))
		for i, a := range assets {
			assetIDs[i] = a.ID
		}
		if err := model.DB.Where("asset_id IN ?", assetIDs).
			Order("sort asc, id asc").Find(&paths).Error; err != nil {
			return nil, err
		}
	}

	// 4. 内存组装:pathByAsset
	pathByAsset := make(map[uint][]model.BusinessAssetPaths, len(assets))
	for _, p := range paths {
		pathByAsset[p.AssetID] = append(pathByAsset[p.AssetID], p)
	}
	// 5. assetByProject
	assetByProject := make(map[uint][]model.BusinessAssets, len(projects))
	for _, a := range assets {
		assetByProject[a.ProjectID] = append(assetByProject[a.ProjectID], a)
	}

	// 6. 输出
	out := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		node := formatProject(p)
		projAssets := assetByProject[p.ID]
		// 统计该项目的"端数量"= 所有资产下路径总条数
		pathCount := 0
		assetNodes := make([]map[string]interface{}, 0, len(projAssets))
		for _, a := range projAssets {
			ap := pathByAsset[a.ID]
			pathCount += len(ap)
			// 路径只展示 name(用户截图要求)
			pathNodes := make([]map[string]interface{}, 0, len(ap))
			for _, p := range ap {
				pathNodes = append(pathNodes, map[string]interface{}{
					"id":   p.ID,
					"name": p.Name,
				})
			}
			assetNodes = append(assetNodes, map[string]interface{}{
				"id":      a.ID,
				"name":    a.Name,
				"address": a.Address,
				"platform": a.Platform,
				"os":      a.OS,
				"paths":   pathNodes,
			})
		}
		node["path_count"] = pathCount
		node["assets"] = assetNodes
		out[i] = node
	}
	return out, nil
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

// SyncResult JMS 同步结果统计
type SyncResult struct {
	Added      int `json:"added"`       // 新增(INSERT)
	Updated    int `json:"updated"`     // 更新(改了 name)
	Removed    int `json:"removed"`     // 软删(JMS 已下架)
	TotalJMS   int `json:"total_jms"`   // JMS 过滤后返回的有效节点数
	TotalLocal int `json:"total_local"` // 同步前本地表非软删记录数
}

// SyncFromJMS 从 JumpServer 同步业务项目(增量 upsert)
//
// 策略(按 code = JMS meta.data.id 匹配):
//
//	JMS 里有,本地没有   → INSERT
//	JMS 里有,本地也有   → 名字变了则 UPDATE name(可能 JMS 改了名)
//	JMS 里没有,本地有   → 软删(DeletedAt 置当前)
//
// 过滤规则(顺序很重要,先粗后细):
//   - 跳过 meta.type != "node"(非组织节点,本接口基本不返回,留作防御)
//   - 跳过 pId == "" 的根节点(树根"Default"在 zTree 里 pId 是空串)
//     ⚠ 不能用 meta.data.id 是否为空判断根节点,根节点也有 data.id(是个 UUID)
//
// 注意:
//   - 本函数会改 code,所以 business_projects.code 唯一索引会被利用做匹配
//   - Description / Sort / Status 不会被 JMS 覆盖(保留人工设置)
//   - 整个过程在一个事务里,失败回滚
func (s *BusinessProjectService) SyncFromJMS(client *jumpserver.Client) (*SyncResult, error) {
	nodes, err := client.GetNodeTree()
	if err != nil {
		return nil, err
	}

	// 1. 过滤有效节点
	type kv struct{ code, name string }
	valid := make([]kv, 0, len(nodes))
	for _, n := range nodes {
		if n.Meta.Type != "node" {
			continue
		}
		// 根节点 pId 是空串(子节点的 pId 都是 "1" 即根的 zTree id)
		if n.PID == "" {
			continue
		}
		code := n.Meta.Data.ID
		if code == "" {
			continue
		}
		valid = append(valid, kv{code: code, name: stripJMSCountSuffix(n.Name)})
	}

	// 2. 拉当前表所有非软删的(code, id, name)
	type row struct {
		ID   uint
		Code string
		Name string
	}
	var existing []row
	if err := model.DB.Model(&model.BusinessProjects{}).
		Select("id, code, name").
		Scan(&existing).Error; err != nil {
		return nil, err
	}
	existingByCode := make(map[string]row, len(existing))
	for _, r := range existing {
		existingByCode[r.Code] = r
	}

	jmsByCode := make(map[string]string, len(valid))
	for _, v := range valid {
		jmsByCode[v.code] = v.name
	}

	result := &SyncResult{
		TotalJMS:   len(valid),
		TotalLocal: len(existing),
	}

	// 3. 事务:INSERT / UPDATE / 软删
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	for code, name := range jmsByCode {
		if r, ok := existingByCode[code]; ok {
			// 已存在 → 仅在 name 变了时 UPDATE
			if r.Name != name {
				if err := tx.Model(&model.BusinessProjects{}).
					Where("id = ?", r.ID).
					Update("name", name).Error; err != nil {
					return nil, err
				}
				result.Updated++
			}
		} else {
			// 新增
			proj := model.BusinessProjects{
				Code:   code,
				Name:   name,
				Status: 1,
				Sort:   0,
			}
			if err := tx.Create(&proj).Error; err != nil {
				return nil, err
			}
			result.Added++
		}
	}

	// 4. 软删:JMS 里没有的本地项目
	for code, r := range existingByCode {
		if _, ok := jmsByCode[code]; !ok {
			if err := tx.Delete(&model.BusinessProjects{}, r.ID).Error; err != nil {
				return nil, err
			}
			result.Removed++
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	committed = true
	return result, nil
}
