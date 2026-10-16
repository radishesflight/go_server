// Package service operation_log.go - 业务操作日志
//
// 统一记 4 类业务事件(部署/上传/删除/同步/下载)到 go_operation_log 表
// 用于审计 + 复盘,前端「部署日志」菜单展示
//
// 2026-10-16 新增
package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"go_server/internal/model"
)

// timeNow 当前时间(包内 alias,便于将来单测 mock)
func timeNow() time.Time { return time.Now() }

// unixMilliToTime unix 毫秒 → time.Time
func unixMilliToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

// 标准事件类型常量(写入 DB 的 event 字段)
const (
	EventDeploy     = "deploy"      // POST /api/codeDeploy/packages/:id/pull
	EventUpload     = "upload"      // POST /api/codeDeploy/packages
	EventDelete     = "delete"      // DELETE /api/codeDeploy/packages/:id
	EventDownload   = "download"    // GET  /api/codeDeploy/packages/:id/download-url
	EventProjectSync = "project.sync" // POST /api/codeDeploy/projects/sync
	EventAssetSync  = "asset.sync"  // POST /api/codeDeploy/assets/sync
	EventSyncAll    = "sync.all"    // POST /api/codeDeploy/sync
)

// TargetType 目标类型
const (
	TargetPackage = "package"
	TargetProject = "project"
	TargetAsset   = "asset"
)

// Result 标记
const (
	ResultSuccess = 1
	ResultFail    = 0
)

// JMSOutputTailLen 远端 stdout/stderr 截取长度
const JMSOutputTailLen = 2000

// LogOperationParams 写日志需要的参数(可选,只填有值的字段)
type LogOperationParams struct {
	Event       string
	Result      int    // ResultSuccess / ResultFail
	Method      string
	Path        string
	TargetType  string
	TargetID    uint
	TargetName  string
	StartedAt   int64  // unix milli,可选
	DurationMS  int64  // 毫秒
	JMSTaskID   string // deploy 时填
	JMSAssetName string // deploy 时填
	JMSOutput   string // deploy 时填,会截到 JMSOutputTailLen
	ErrorMsg    string // 失败原因
}

// LogOperation 异步写一条操作日志
//
// 为什么 async(service 立即返回,后台 goroutine 写 DB):
//   1. 写日志不应该拖慢业务接口(几十毫秒级)
//   2. 写日志失败不应该影响业务结果
//   3. 业务接口已经返回,日志晚几毫秒落库无影响
//
// 入参 c 是 gin.Context(传 nil 也能跑,只是拿不到 user/username/ip/ua)
func LogOperation(c *gin.Context, p LogOperationParams) {
	if p.Event == "" {
		return // 至少要事件类型
	}
	go func() {
		log := &model.OperationLog{
			Event:     p.Event,
			Result:    p.Result,
			Method:    p.Method,
			Path:      p.Path,
			TargetType: p.TargetType,
			TargetID:  p.TargetID,
			TargetName: p.TargetName,
			DurationMS: p.DurationMS,
			JMSTaskID: p.JMSTaskID,
			JMSAssetName: p.JMSAssetName,
			JMSOutput: trimTail(p.JMSOutput, JMSOutputTailLen),
			ErrorMsg: trimTail(p.ErrorMsg, 500),
		}
		if c != nil {
			if v, ok := c.Get("user_id"); ok {
				if uid, ok2 := v.(uint); ok2 {
					log.UserID = uid
				}
			}
			if v, ok := c.Get("username"); ok {
				if s, ok2 := v.(string); ok2 {
					log.Username = s
				}
			}
			log.ClientIP = c.ClientIP()
			ua := c.Request.UserAgent()
			if len(ua) > 200 {
				ua = ua[:200]
			}
			log.UserAgent = ua
		}
		if p.StartedAt > 0 {
			t := unixMilliToTime(p.StartedAt)
			log.StartedAt = t
		} else {
			log.StartedAt = timeNow()
		}
		if err := model.DB.Create(log).Error; err != nil {
			// 写失败只 log,不暴露给调用方(业务已完成)
			// 实际生产可用 logger.L.Error,但本包 import logger 会循环,先不引
			// logger.L.Error("write operation log failed", zap.Error(err))
			_ = err
		}
	}()
}

// trimTail 截取字符串尾部 n 字符(防超字段长度)
func trimTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
