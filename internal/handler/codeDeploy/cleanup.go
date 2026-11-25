// Package codeDeploy cleanup.go - 一键清除部署数据 HTTP 入口
//
// 接口:
//
//	POST /api/codeDeploy/operation-logs/clear  按时间范围清除部署日志 + 代码包(含对象存储文件)
//
// 请求体:
//
//	{
//	  "start": "2026-11-01",   // 必填,含当天
//	  "end":   "2026-11-24",   // 必填,含当天
//	  "clear_logs":     true,  // 可选,默认 true
//	  "clear_packages": true   // 可选,默认 true
//	}
//
// 响应 data:
//
//	{
//	  "logs_deleted": 12,
//	  "packages_deleted": 3,
//	  "objects_deleted": 3,
//	  "failed_objects": [{"file_path": "code_packages/x.zip", "error": "..."}]
//	}
package codeDeploy

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

var cleanupSvc = service.NewDeployCleanupService()

// 清除请求参数
type clearDeployDataReq struct {
	Start         string `json:"start"`
	End           string `json:"end"`
	ClearLogs     *bool  `json:"clear_logs"`
	ClearPackages *bool  `json:"clear_packages"`
}

// ClearDeployData 一键清除部署数据
// POST /api/codeDeploy/operation-logs/clear
func ClearDeployData(c *gin.Context) {
	startedAt := time.Now()

	var req clearDeployDataReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	rangeStart, rangeEnd, err := parseCleanupRange(req.Start, req.End)
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, err.Error())
		return
	}

	opts := service.CleanupOptions{
		Start:         rangeStart,
		End:           rangeEnd,
		ClearLogs:     true,
		ClearPackages: true,
	}
	if req.ClearLogs != nil {
		opts.ClearLogs = *req.ClearLogs
	}
	if req.ClearPackages != nil {
		opts.ClearPackages = *req.ClearPackages
	}
	if !opts.ClearLogs && !opts.ClearPackages {
		handler.Error(c, handler.CodeParamsInvalid, "请至少选择一项清除内容")
		return
	}

	result, err := cleanupSvc.Clear(opts)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "清除失败: "+err.Error())
		return
	}

	// 清除动作本身留痕(写在删除之后,不会被本次清除掉)
	service.LogOperation(c, service.LogOperationParams{
		Event:      service.EventClear,
		Result:     service.ResultSuccess,
		Method:     c.Request.Method,
		Path:       c.FullPath(),
		TargetType: service.TargetCleanup,
		TargetName: fmt.Sprintf("%s ~ %s", req.Start, req.End),
		DurationMS: time.Since(startedAt).Milliseconds(),
	})

	handler.Success(c, result)
}

// parseCleanupRange 解析清除时间范围,统一成 [start, end) 区间
// 结束日期只给到天时,自动补到次日 0 点(含当天)
func parseCleanupRange(startStr, endStr string) (time.Time, time.Time, error) {
	startStr = strings.TrimSpace(startStr)
	endStr = strings.TrimSpace(endStr)
	if startStr == "" || endStr == "" {
		return time.Time{}, time.Time{}, errors.New("请选择清除的时间范围")
	}

	rangeStart, err := parseCleanupTime(startStr, false)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("开始日期格式错误")
	}
	rangeEnd, err := parseCleanupTime(endStr, true)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("结束日期格式错误")
	}
	if !rangeStart.Before(rangeEnd) {
		return time.Time{}, time.Time{}, errors.New("开始日期必须早于结束日期")
	}
	return rangeStart, rangeEnd, nil
}

// parseCleanupTime 解析单个时间点
// 只给到天(yyyy-MM-dd)时按本地时区解析,避免 UTC 偏移把当天早上 8 小时切掉
func parseCleanupTime(s string, isEnd bool) (time.Time, error) {
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		if isEnd {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
