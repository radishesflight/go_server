// Package service business_asset_service.go - 业务项目下的资产
//
// 从 JMS /api/v1/assets/assets/ 拉取资产列表,按 nodes[0] 关联到 business_projects
//
// 数据流(两段同步):
//  1. 同步节点(已有 SyncFromJMS,只拉节点树,无资产)
//  2. 同步资产(本服务,按 project_id 关联到节点)
//
// 一致性:assets 同步依赖 projects 已存在;通常先跑项目同步,再跑资产同步
package service

import (
	"errors"

	"go_server/internal/model"
	"go_server/pkg/jumpserver"
)

// 业务错误(给 handler 层翻译业务码)
var (
	ErrAssetNotFound = errors.New("业务资产不存在")
	ErrAssetSyncFail = errors.New("同步资产失败")
)

// AssetSyncResult 资产同步结果统计
type AssetSyncResult struct {
	Added      int `json:"added"`       // 新增(INSERT)
	Updated    int `json:"updated"`     // 更新(name/address/platform 变了)
	Removed    int `json:"removed"`     // 软删(JMS 已下架)
	TotalJMS   int `json:"total_jms"`   // JMS 返回的有效资产数
	TotalLocal int `json:"total_local"` // 同步前本地非软删资产数
}

// BusinessAssetService 业务资产服务
type BusinessAssetService struct{}

// NewBusinessAssetService 构造
func NewBusinessAssetService() *BusinessAssetService { return &BusinessAssetService{} }

// formatAsset 把 model 序列化成前端 map
func formatAsset(a model.BusinessAssets) map[string]interface{} {
	return map[string]interface{}{
		"id":         a.ID,
		"project_id": a.ProjectID,
		"jms_id":     a.JmsID,
		"name":       a.Name,
		"address":    a.Address,
		"platform":   a.Platform,
		"os":         a.OS,
		"protocol":   a.Protocol,
		"port":       a.Port,
		"comment":    a.Comment,
		"status":     a.Status,
		"created_at": model.FormatTime(a.CreatedAt),
		"updated_at": model.FormatTime(a.UpdatedAt),
	}
}

// ListByProject 查某项目下的所有资产(包含已禁用的,供详情页用)
func (s *BusinessAssetService) ListByProject(projectID uint) ([]map[string]interface{}, error) {
	if projectID == 0 {
		return nil, ErrAssetNotFound
	}
	var assets []model.BusinessAssets
	if err := model.DB.Where("project_id = ?", projectID).
		Order("id asc").Find(&assets).Error; err != nil {
		return nil, ErrAssetSyncFail
	}
	out := make([]map[string]interface{}, len(assets))
	for i, a := range assets {
		out[i] = formatAsset(a)
	}
	return out, nil
}

// projectCodeToIDMap 查 (code, id) 映射,用于把 JMS asset.nodes[0].id 关联到本地 project.id
func (s *BusinessAssetService) projectCodeToIDMap() (map[string]uint, error) {
	var projects []model.BusinessProjects
	if err := model.DB.Select("id, code").Find(&projects).Error; err != nil {
		return nil, err
	}
	m := make(map[string]uint, len(projects))
	for _, p := range projects {
		m[p.Code] = p.ID
	}
	return m, nil
}

// SyncFromJMS 从 JMS 同步资产(全量 + 分页,只读 GET)
//
// 策略(按 jms_id 匹配):
//   - JMS 有,本地有 → UPDATE name/address/platform
//   - JMS 有,本地没有 → INSERT(通过 nodes[0].id 关联到 project_id)
//   - JMS 没有,本地有 → 软删(deleted_at = NOW)
//
// 全部在一个事务里,任一失败回滚
func (s *BusinessAssetService) SyncFromJMS(client *jumpserver.Client) (*AssetSyncResult, error) {
	// 1. 项目 code → id 映射(把 JMS 的 node uuid 翻译成本地 project_id)
	codeMap, err := s.projectCodeToIDMap()
	if err != nil {
		return nil, err
	}

	// 2. 分页拉 JMS 资产
	allAssets := make([]jumpserver.Asset, 0, 256)
	url := ""
	for {
		resp, err := client.ListAssets(url)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, resp.Results...)

		// next 是完整 URL(可能含其他 query),到 nil 停
		if resp.Next == nil || *resp.Next == "" {
			break
		}
		url = *resp.Next
	}

	// 3. 拉本地现有非软删资产
	type row struct {
		ID    uint
		JmsID string
		Name  string
	}
	var existing []row
	if err := model.DB.Model(&model.BusinessAssets{}).
		Select("id, jms_id, name").
		Where("deleted_at IS NULL").
		Scan(&existing).Error; err != nil {
		return nil, err
	}
	existingByJmsID := make(map[string]row, len(existing))
	for _, r := range existing {
		existingByJmsID[r.JmsID] = r
	}

	// 4. 过滤有效资产:有 nodes[0] 且 codeMap 里有这个 project code
	type validAsset struct {
		jmsID     string
		projectID uint
		name      string
		address   string
		platform  string
		os        string
		protocol  string
		port      int
		comment   string
	}
	valid := make([]validAsset, 0, len(allAssets))
	for _, a := range allAssets {
		if len(a.Nodes) == 0 {
			continue
		}
		// 取第一个 node 作为归属
		// JMS 资产接口(/api/v1/assets/assets/) 的 nodes[0].id 直接是项目 UUID
		// (不是 zTree "1:17" 内部 ID),所以这里用 firstNode.ID,不是 firstNode.Meta.Data.ID
		firstNode := a.Nodes[0]
		nodeKey := firstNode.ID
		if nodeKey == "" {
			continue
		}
		// 找本地 project
		pid, ok := codeMap[nodeKey]
		if !ok {
			// JMS 节点不在本地业务项目里 → 跳过(可能项目未同步)
			continue
		}
		platform := ""
		os := ""
		if a.Platform != nil {
			platform = a.Platform.Name
			os = platform
		}
		// 只同步 platform = Linux 的资产
		// 业务侧:SSH 远程命令执行只需要 Linux 资产,MySQL 数据库由其他渠道管理
		if platform != "Linux" {
			continue
		}
		protoName := ""
		protoPort := 0
		if len(a.Protocols) > 0 {
			protoName = a.Protocols[0].Name
			protoPort = a.Protocols[0].Port
		}
		valid = append(valid, validAsset{
			jmsID:     a.ID,
			projectID: pid,
			name:      a.Name,
			address:   a.Address,
			platform:  platform,
			os:        os,
			protocol:  protoName,
			port:      protoPort,
			comment:   a.Comment,
		})
	}

	// 5. 事务:INSERT / UPDATE / 软删
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

	result := &AssetSyncResult{
		TotalJMS:   len(allAssets),
		TotalLocal: len(existing),
	}

	// 按 jms_id 去重
	jmsIDSet := make(map[string]bool, len(valid))
	for _, v := range valid {
		jmsIDSet[v.jmsID] = true
		if r, ok := existingByJmsID[v.jmsID]; ok {
			// 已存在 → name/address 变了才 UPDATE
			if r.Name != v.name {
				if err := tx.Model(&model.BusinessAssets{}).
					Where("id = ?", r.ID).
					Updates(map[string]interface{}{
						"name":     v.name,
						"address":  v.address,
						"platform": v.platform,
						"protocol": v.protocol,
						"port":     v.port,
						"comment":  v.comment,
					}).Error; err != nil {
					return nil, err
				}
				result.Updated++
			}
		} else {
			// 新增
			asset := model.BusinessAssets{
				ProjectID: v.projectID,
				JmsID:     v.jmsID,
				Name:      v.name,
				Address:   v.address,
				Platform:  v.platform,
				OS:        v.os,
				Protocol:  v.protocol,
				Port:      v.port,
				Comment:   v.comment,
				Status:    1,
			}
			if err := tx.Create(&asset).Error; err != nil {
				return nil, err
			}
			result.Added++
		}
	}

	// 软删:JMS 已没有的本地资产
	for jmsID, r := range existingByJmsID {
		if !jmsIDSet[jmsID] {
			if err := tx.Delete(&model.BusinessAssets{}, r.ID).Error; err != nil {
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
