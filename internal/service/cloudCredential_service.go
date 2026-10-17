// Package service cloudCredential_service.go - 云平台凭证业务层
//
// 2026-10-17 新增,提供 list/get/create/update/delete,以及 secret_key 的 mask 工具。
package service

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"go_server/internal/model"
)

// 业务错误(给 handler 翻译)
var (
	ErrCloudCredentialNotFound   = errors.New("云平台凭证不存在")
	ErrCloudCredentialNameExists = errors.New("同名同类型凭证已存在")
)

// ListCloudCredentials 列出凭证(可选 type 筛选 + 关键字模糊匹配 name/access_key)
//
//   - credType: 空=全部;否则过滤 `type`
//   - keyword:  空=全部;否则 name LIKE / access_key LIKE
func ListCloudCredentials(credType, keyword string) ([]model.CloudCredential, int64, error) {
	db := model.DB.Model(&model.CloudCredential{}).Where("deleted_at IS NULL")
	if credType != "" {
		db = db.Where("type = ?", credType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR access_key LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.CloudCredential
	if err := db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetCloudCredential 单条详情(secret_key 明文,只给详情页用)
func GetCloudCredential(id uint64) (*model.CloudCredential, error) {
	var c model.CloudCredential
	if err := model.DB.Where("id = ?", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCloudCredentialNotFound
		}
		return nil, err
	}
	return &c, nil
}

// CreateCloudCredential 新增
func CreateCloudCredential(c *model.CloudCredential) error {
	// 同名同类型唯一
	var existing model.CloudCredential
	err := model.DB.Where("name = ? AND type = ?", c.Name, c.Type).First(&existing).Error
	if err == nil {
		return ErrCloudCredentialNameExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return model.DB.Create(c).Error
}

// UpdateCloudCredential 编辑(允许保留 name+type 不变)
//
//	不用 Save()(它会带 created_at=0 值覆盖,违反 strict mode 的 DATETIME NOT NULL 规则),
//	改用 Updates + Select 显式列,只更新业务列,created_at 由 GORM 维持不变。
func UpdateCloudCredential(id uint64, c *model.CloudCredential) error {
	var existing model.CloudCredential
	if err := model.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCloudCredentialNotFound
		}
		return err
	}
	// 重名检查:如果 name 或 type 改了,看是否撞别的行
	if existing.Name != c.Name || existing.Type != c.Type {
		var dup model.CloudCredential
		err := model.DB.Where("name = ? AND type = ? AND id <> ?", c.Name, c.Type, id).First(&dup).Error
		if err == nil {
			return ErrCloudCredentialNameExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	// 只更新业务字段(隐式跳过 created_at),UpdatedAt 由 GORM 自动 set
	return model.DB.Model(&existing).
		Select("name", "type", "access_key", "secret_key", "remark").
		Updates(c).Error
}

// DeleteCloudCredential 删除(软删)
func DeleteCloudCredential(id uint64) error {
	res := model.DB.Delete(&model.CloudCredential{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCloudCredentialNotFound
	}
	return nil
}

// MaskSecretKey 列表展示用,secret 只露后 4 位(防截屏泄露)
func MaskSecretKey(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

// CloudCredentialListItem 列表返回值(secret_key 已被 mask)
type CloudCredentialListItem struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	AccessKey string    `json:"access_key"`
	SecretKey string    `json:"secret_key"` // 已 mask
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MaskCredentialList 把 CloudCredential 列表转成 mask 后的展示结构
func MaskCredentialList(list []model.CloudCredential) []CloudCredentialListItem {
	out := make([]CloudCredentialListItem, 0, len(list))
	for _, c := range list {
		out = append(out, CloudCredentialListItem{
			ID:        c.ID,
			Name:      c.Name,
			Type:      c.Type,
			AccessKey: c.AccessKey,
			SecretKey: MaskSecretKey(c.SecretKey),
			Remark:    c.Remark,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return out
}
