// Package service business_asset_path_service.go - 业务资产的 Linux 目录路径
//
// 作用:为每个 Linux 资产配置 1..N 个目录路径(后续 SSH 任务可选 cwd)
// 本服务**只描述意图**,不实际执行命令(遵循"JMS 生产环境禁命令"原则)
package service

import (
	"errors"
	"strings"

	"go_server/internal/model"
)

// 业务错误
var (
	ErrAssetPathNotFound  = errors.New("资产路径不存在")
	ErrAssetPathNameEmpty = errors.New("路径名称不能为空")
	ErrAssetPathPathEmpty = errors.New("路径不能为空")
)

// AssetPathService 资产路径服务
type AssetPathService struct{}

// NewAssetPathService 构造
func NewAssetPathService() *AssetPathService { return &AssetPathService{} }

// ListByAsset 查某资产下的所有路径(按 sort asc, id asc)
//
//	pathType 过滤:"" 不过滤;"api" 只返 type='api'; "web" 只返 type='web'; "other"/"apk"/"ios" 同理
//	(2026-10-16 新增:为 Online.vue "后端API部署" 菜单只展示 type=api 的路径,
//	 2026-10-16 扩展 apk/ios:为 APK包管理 菜单展示 type IN ('apk','ios') 的路径)
func (s *AssetPathService) ListByAsset(assetID uint, pathType string) ([]map[string]interface{}, error) {
	if assetID == 0 {
		return nil, ErrAssetPathNotFound
	}
	var paths []model.BusinessAssetPaths
	db := model.DB.Where("asset_id = ?", assetID)
	if pathType != "" {
		// 兼容:老数据 type=NULL 不在过滤范围内(用户说"老的不考虑")
		db = db.Where("type = ?", pathType)
	}
	if err := db.Order("sort asc, id asc").Find(&paths).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(paths))
	for i, p := range paths {
		out[i] = formatAssetPath(p)
	}
	return out, nil
}

func formatAssetPath(p model.BusinessAssetPaths) map[string]interface{} {
	return map[string]interface{}{
		"id":         p.ID,
		"asset_id":   p.AssetID,
		"name":       p.Name,
		"path":       p.Path,
		"type":       p.Type,
		"sort":       p.Sort,
		"created_at": model.FormatTime(p.CreatedAt),
		"updated_at": model.FormatTime(p.UpdatedAt),
	}
}

// AddPath 给资产加一条路径(2026-10-16 加 type 参数,v2 string + DB default)
//
//	type 接收 string(前端 el-select value):"api" / "web" / "other" / "apk" / "ios"
//	空 / 不传 → 不赋 p.Type,让 DB DEFAULT 'web' 兜底(GORM 不写零值字段,触发 default)
//	非法值(非 api/web/other/apk/ios)→ 归 "other"
//
//	path 必填规则(2026-10-16 扩展):
//	  - type IN ("api", "web") → path 必填(这两个类型真部署到服务器路径)
//	  - type IN ("apk", "ios", "other") → path 可留空(只挂代码包不部署)
func (s *AssetPathService) AddPath(assetID uint, name, path, pathType string, sort int) (map[string]interface{}, error) {
	if assetID == 0 {
		return nil, ErrAssetPathNotFound
	}
	if name == "" {
		return nil, ErrAssetPathNameEmpty
	}
	// 资产必须存在且软删 status=1
	var asset model.BusinessAssets
	if err := model.DB.First(&asset, assetID).Error; err != nil {
		return nil, ErrAssetPathNotFound
	}
	p := model.BusinessAssetPaths{
		AssetID: assetID,
		Name:    name,
		Path:    path,
		Sort:    sort,
	}
	// type 显式传:校验白名单,非法值归 other
	if pathType != "" {
		switch pathType {
		case "api", "web", "other", "apk", "ios":
			// ok
		default:
			pathType = "other"
		}
		p.Type = pathType
	}
	// path 必填校验:api/web 真部署必须填;apk/ios/other 可空
	if p.Type == "api" || p.Type == "web" {
		if strings.TrimSpace(p.Path) == "" {
			return nil, ErrAssetPathPathEmpty
		}
	}
	// pathType == "" 时 p.Type 保持空字符串,GORM Insert 会跳过,DB DEFAULT 'web' 触发
	if err := model.DB.Create(&p).Error; err != nil {
		return nil, err
	}
	return formatAssetPath(p), nil
}

// DeletePath 删一条路径(真删,业务表约定不软删)
func (s *AssetPathService) DeletePath(id uint) error {
	if id == 0 {
		return ErrAssetPathNotFound
	}
	return model.DB.Unscoped().Delete(&model.BusinessAssetPaths{}, id).Error
}
