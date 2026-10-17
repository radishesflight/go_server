// Package service cloudServer_service.go - 云服务器持久化业务层
//
// 2026-10-17 新增,与云平台凭证(CloudCredential)联动:
//
//	每条服务器记录挂在某条 credential 下,前端能按 credential_id / vendor 筛选,
//	多账号场景可以各自拉各自的服务器列表。
//
// 同步流程:
//  1. service.SyncCloudServers(credentialID=0):遍历所有 aliyun 类型凭证,逐个 sync
//  2. service.SyncCloudServers(credentialID=N>0):只同步那一条凭证
//  3. 对每个凭证,基于它的 AK/SK 创建阿里云 client,跨 33 region 拉 DescribeInstances
//  4. hard-delete 该凭证名下旧数据 + 批量 insert 新数据
//
// 暂仅支持阿里云(2026-10-17,腾讯云 v3 签名待后续接入)。
package service

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"go_server/internal/model"
	ecspkg "go_server/pkg/aliyun/ecs"

	"gorm.io/gorm"
)

// 业务错误
var (
	ErrCloudServerNoCredentials   = errors.New("没有可同步的阿里云凭证")
	ErrCloudServerSyncUnsupported = errors.New("该厂商暂不支持同步(目前仅支持阿里云)")
)

// 同步状态(进程内单例,前端可轮询)
type serverSyncStatus struct {
	running   atomic.Bool
	lastAt    atomic.Int64
	lastCount atomic.Int64
	lastError atomic.Pointer[string]
}

var globalServerSyncStatus serverSyncStatus

// ServerSyncStatusResp 同步状态响应
type ServerSyncStatusResp struct {
	Running   bool   `json:"running"`
	LastAt    int64  `json:"last_at"`
	LastCount int64  `json:"last_count"`
	LastError string `json:"last_error"`
}

func GetServerSyncStatus() ServerSyncStatusResp {
	resp := ServerSyncStatusResp{
		Running:   globalServerSyncStatus.running.Load(),
		LastAt:    globalServerSyncStatus.lastAt.Load(),
		LastCount: globalServerSyncStatus.lastCount.Load(),
	}
	if p := globalServerSyncStatus.lastError.Load(); p != nil {
		resp.LastError = *p
	}
	return resp
}

// ListCloudServers 列表查询(支持 vendor / credential_id / region / keyword 过滤)
func ListCloudServers(credentialID uint64, vendor, region, keyword string) ([]model.CloudServer, int64, error) {
	db := model.DB.Model(&model.CloudServer{}).Where("deleted_at IS NULL")
	if credentialID > 0 {
		db = db.Where("credential_id = ?", credentialID)
	}
	if vendor != "" {
		db = db.Where("vendor = ?", vendor)
	}
	if region != "" {
		db = db.Where("region = ?", region)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR instance_id LIKE ? OR private_ip LIKE ? OR public_ip LIKE ?",
			like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.CloudServer
	if err := db.Order("vendor ASC, region ASC, instance_id ASC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// SyncCloudServers 触发一次同步(2026-10-17)
//
//   - credentialID == 0 → 同步全部 aliyun 凭证
//   - credentialID >  0 → 同步指定凭证
//
// 返回:同步成功的凭证数、写入的服务器总数、最后一个错误(若有)
func SyncCloudServers(credentialID uint64) (syncStatus string, totalCount int, err error) {
	// process 内并发锁(避免用户连点同步按钮产生并发同步)
	if !globalServerSyncStatus.running.CompareAndSwap(false, true) {
		return "已有同步任务在跑,请稍候", 0, errors.New("已有同步任务在跑")
	}
	defer globalServerSyncStatus.running.Store(false)

	// 取要同步的凭证列表
	var creds []model.CloudCredential
	q := model.DB.Where("deleted_at IS NULL").Where("type = ?", model.CloudCredentialTypeAliyun)
	if credentialID > 0 {
		q = q.Where("id = ?", credentialID)
	}
	if err := q.Find(&creds).Error; err != nil {
		setSyncErr("查询凭证失败: " + err.Error())
		return "查询凭证失败", 0, err
	}
	if len(creds) == 0 {
		setSyncErr("没有可同步的阿里云凭证")
		return "没有可同步的阿里云凭证", 0, ErrCloudServerNoCredentials
	}

	totalCount = 0
	var lastErr error
	now := time.Now()

	for i := range creds {
		cred := &creds[i]
		c := ecspkg.NewClient(cred.AccessKey, cred.SecretKey, "cn-hangzhou", "aliyuncs.com")
		inst, regErrs, err := c.DescribeInstancesAllRegions()
		if err != nil {
			// 整 credential 拉取顶层失败(可能是 AK 错、网络全挂)
			msg := "凭证 '" + cred.Name + "' 拉取失败: " + err.Error()
			setSyncErr(msg)
			lastErr = errors.New(msg)
			continue
		}
		if len(regErrs) > 0 {
			// 部分 region 失败,继续写成功的
			setSyncErr("凭证 '" + cred.Name + "' 部分 region 失败(共 " +
				itoa(len(regErrs)) + " 个);后续以成功的写入为准")
		}
		count, werr := writeServersForCredential(cred.ID, inst, now)
		if werr != nil {
			msg := "凭证 '" + cred.Name + "' 写入失败: " + werr.Error()
			setSyncErr(msg)
			lastErr = errors.New(msg)
			continue
		}
		totalCount += count
	}

	globalServerSyncStatus.lastAt.Store(time.Now().Unix())
	globalServerSyncStatus.lastCount.Store(int64(totalCount))
	if lastErr == nil {
		globalServerSyncStatus.lastError.Store(nil)
	} else {
		// 保留 lastError 让前端能看到(后面会被下次成功清掉)
		s := lastErr.Error()
		globalServerSyncStatus.lastError.Store(&s)
	}
	return "ok", totalCount, lastErr
}

func setSyncErr(msg string) {
	globalServerSyncStatus.lastError.Store(&msg)
}

// writeServersForCredential 把单个 credential 拉的 instance 列表硬删 + 批量插新
func writeServersForCredential(credID uint64, instances []ecspkg.Instance, snapshotAt time.Time) (int, error) {
	// 硬删该 credential 下的旧记录(同 credential 下 UQ (vendor, instance_id) 会冲突)
	if err := model.DB.Unscoped().Where("credential_id = ?", credID).Delete(&model.CloudServer{}).Error; err != nil {
		return 0, err
	}
	if len(instances) == 0 {
		return 0, nil
	}
	rows := make([]model.CloudServer, 0, len(instances))
	for _, ins := range instances {
		cs := model.CloudServer{
			CredentialID: credID,
			Vendor:       model.CloudServerVendorAliyun,
			InstanceID:   ins.InstanceID,
			Name:         ins.InstanceName,
			Region:       ins.RegionID,
			Zone:         ins.ZoneID,
			InstanceType: ins.InstanceType,
			Status:       ins.Status,
			PrivateIP:    ins.PrivateIP,
			PublicIP:     ins.PublicIP,
			OSName:       ins.OSName,
			CPU:          ins.CPU,
			MemoryGB:     ins.Memory / 1024, // 阿里云 API 返回 MiB,DB 用 GB 不直观存 MiB,这里统一转
			SnapshotAt:   snapshotAt,
		}
		// instance_created_at / expired_at 解析 ISO8601
		cs.InstanceCreatedAt = parseAliyunTimePtr(ins.CreationTime)
		cs.ExpiredAt = parseAliyunTimePtr(ins.ExpiredTime)
		rows = append(rows, cs)
	}
	// 分批插
	const batch = 100
	for i := 0; i < len(rows); i += batch {
		end := i + batch
		if end > len(rows) {
			end = len(rows)
		}
		if err := model.DB.Create(rows[i:end]).Error; err != nil {
			return i, err
		}
	}
	return len(rows), nil
}

// parseAliyunTimePtr 把阿里云 ISO8601 字符串(空时返 nil)转成 *time.Time
//
// 阿里云 CreationTime / ExpiredTime 实际形态:
//   - "2026-08-12T08:52Z"        短写(只有 HH:MM,无秒)
//   - "2026-08-12T08:52:31Z"     标准 ISO8601 UTC
//   - "2026-09-02T16:00Z"        短写
//
// Go 的 time.RFC3339 layout = "2006-01-02T15:04:05Z07:00",要求"秒"字段必须存在,
// 所以短写 "T08:52Z" 直接 Parse 会报 `cannot parse "Z" as ":"`(layout 第 6 位是 `:` 占位,
// 但输入已经走到字符串末尾、没有 `:`)。workaround:对短写补 `:00` 秒后正常 Parse。
//
// 其它失败场景(意外格式/无时区)直接返 nil,前端按"创建时间未知"展示,不阻断同步。
func parseAliyunTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	// 短写 "...THH:MMZ" 补秒成标准 RFC3339
	if strings.HasSuffix(s, "Z") {
		// T 之后第一个 ':' 到 Z 之前,数冒号个数:2 个 = 有秒,1 个 = 缺秒
		tIdx := strings.Index(s, "T")
		if tIdx >= 0 {
			rest := s[tIdx:]
			// 只在第一段 HH:MM 时补秒(也覆盖未来有时区格式 "T08:52+0800")
			if strings.Count(rest, ":") == 1 {
				s = s[:len(s)-1] + ":00Z"
			}
		}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}

// itoa 避免引入 strconv 的循环(微优化)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// 显式占位让 gorm.ErrRecordNotFound 用上,避免 go vet 报 unused import
var _ = gorm.ErrRecordNotFound
