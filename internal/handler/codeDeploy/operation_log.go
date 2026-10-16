// Package codeDeploy operation_log.go - 业务操作日志 HTTP 入口
//
// 接口:
//
//	GET  /api/codeDeploy/operation-logs            分页列表(支持 event/result/包名/时间筛选)
//	GET  /api/codeDeploy/operation-logs/:id       详情(返回完整字段含 JMS 远端输出)
//
// 2026-10-16 新增:支撑"部署日志"菜单(用户要求详细记录操作日志)
package codeDeploy

import (
	"strconv"
	"strings"
	"time"

	"go_server/internal/handler"
	"go_server/internal/model"

	"github.com/gin-gonic/gin"
)

// ListOperationLogs 分页列表 + 多维筛选
// GET /api/codeDeploy/operation-logs?event=deploy&result=1&keyword=xxx&page=1&size=20&start=2026-10-01&end=2026-10-16
//
// 筛选条件(全部可选):
//   - event     事件类型(精确匹配),多选暂不支持
//   - result    1=成功 / 0=失败
//   - keyword   包名/目标名 LIKE 模糊匹配
//   - start/end ISO 时间字符串,只查 [start, end) 区间
//   - page/size 分页,默认 1/20,size 上限 200
func ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	event := strings.TrimSpace(c.Query("event"))
	resultStr := c.Query("result")
	keyword := strings.TrimSpace(c.Query("keyword"))
	startStr := strings.TrimSpace(c.Query("start"))
	endStr := strings.TrimSpace(c.Query("end"))

	db := model.DB.Model(&model.OperationLog{})
	if event != "" {
		db = db.Where("event = ?", event)
	}
	if resultStr != "" {
		if r, err := strconv.Atoi(resultStr); err == nil {
			db = db.Where("result = ?", r)
		}
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		// 包名(target_name) + 用户名(username) 模糊匹配
		db = db.Where("target_name LIKE ? OR username LIKE ?", like, like)
	}
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			db = db.Where("created_at >= ?", t)
		} else if t, err := time.Parse("2006-01-02", startStr); err == nil {
			db = db.Where("created_at >= ?", t)
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			db = db.Where("created_at < ?", t)
		} else if t, err := time.Parse("2006-01-02", endStr); err == nil {
			// 截止日期加一天(包含 end 当天)
			db = db.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		handler.Error(c, handler.CodeUnknown, "查询日志总数失败")
		return
	}
	var rows []model.OperationLog
	if err := db.Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&rows).Error; err != nil {
		handler.Error(c, handler.CodeUnknown, "查询日志失败")
		return
	}

	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, formatOperationLog(&r))
	}
	handler.Success(c, gin.H{
		"list":  out,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetOperationLog 详情
// GET /api/codeDeploy/operation-logs/:id
func GetOperationLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的日志ID")
		return
	}
	var row model.OperationLog
	if err := model.DB.First(&row, id).Error; err != nil {
		handler.Error(c, handler.CodeProjectNotFound, "日志不存在")
		return
	}
	handler.Success(c, formatOperationLog(&row))
}

// formatOperationLog 序列化成给前端的 map
func formatOperationLog(r *model.OperationLog) map[string]interface{} {
	return map[string]interface{}{
		"id":             r.ID,
		"event":          r.Event,
		"result":         r.Result,
		"method":         r.Method,
		"path":           r.Path,
		"user_id":        r.UserID,
		"username":       r.Username,
		"target_type":    r.TargetType,
		"target_id":      r.TargetID,
		"target_name":    r.TargetName,
		"started_at":     model.FormatTime(r.StartedAt),
		"finished_at":    formatFinishedAt(r.FinishedAt),
		"duration_ms":    r.DurationMS,
		"jms_task_id":    r.JMSTaskID,
		"jms_asset_name": r.JMSAssetName,
		"jms_output":     r.JMSOutput,
		"error_msg":      r.ErrorMsg,
		"client_ip":      r.ClientIP,
		"user_agent":     r.UserAgent,
		"created_at":     model.FormatTime(r.CreatedAt),
	}
}

func formatFinishedAt(t *time.Time) string {
	if t == nil {
		return ""
	}
	return model.FormatTime(*t)
}
