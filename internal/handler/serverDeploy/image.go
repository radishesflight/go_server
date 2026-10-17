// Package serverDeploy image.go - 阿里云 ECS 镜像管理 HTTP 入口
//
// 接口列表(2026-10-17 改造,从实时拉改为 DB 持久化):
//
//	GET    /api/serverDeploy/images?region=&type=&keyword=&page=&size=   列镜像(从本地 DB 查)
//	POST   /api/serverDeploy/images/sync                                     触发同步(拉阿里云 → 删旧 → 插新)
//	GET    /api/serverDeploy/images/sync/status                             查询同步状态
//	GET    /api/serverDeploy/regions                                        列已同步的 region(DB 查,代替实时拉)
//
// 业务码:
//
//	service.ErrAliyunImageNotConfig → CodeAliyunECSNotConfig(9001)
package serverDeploy

import (
	"errors"
	"strconv"
	"time"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// 业务码(2026-10-17 新增,9001 起避免冲突)
const (
	CodeAliyunECSNotConfig = 9001
)

// ListImages GET /api/serverDeploy/images
//
// 从本地 DB 查(2026-10-17 改造),毫秒级返回,支持:
//   - region     区域筛选(空=全部)
//   - type       镜像类型筛选(Self / Others / 空=全部)
//   - keyword    模糊匹配 image_id / image_name
//   - page/size  分页
func ListImages(c *gin.Context) {
	region := c.Query("region")
	imageType := c.Query("type")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))

	// 简单分页(数据量小,本地分页即可)
	list, total, err := service.ListImagesFromDB(region, imageType, keyword)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询镜像失败: "+err.Error())
		return
	}
	// 切片
	start := (page - 1) * size
	end := start + size
	if start > len(list) {
		start = len(list)
	}
	if end > len(list) {
		end = len(list)
	}
	paged := list[start:end]

	handler.Success(c, gin.H{
		"list":  paged,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// ListRegions GET /api/serverDeploy/regions
//
// 列出本地 DB 里已同步的 region(代替实时调 DescribeRegions)
func ListRegions(c *gin.Context) {
	regions, err := service.ListImageRegionsFromDB()
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询 region 失败: "+err.Error())
		return
	}
	out := make([]gin.H, 0, len(regions))
	for _, r := range regions {
		out = append(out, gin.H{"RegionId": r, "LocalName": r})
	}
	handler.Success(c, gin.H{"list": out})
}

// SyncImages POST /api/serverDeploy/images/sync
//
// 触发一次全量同步:拉阿里云 → 删旧 → 插新。
// 耗时约 12 秒(33 region 并发),期间同进程内并发请求会被拒(同步任务锁)。
//
// 响应:同步完成后返回新写入的行数 + 耗时。
func SyncImages(c *gin.Context) {
	start := time.Now()
	err := service.SyncImages()
	elapsed := time.Since(start)
	if err != nil {
		translateSyncErr(c, err)
		return
	}
	st := service.GetSyncStatus()
	handler.Success(c, gin.H{
		"message":    "同步完成",
		"elapsed_ms": elapsed.Milliseconds(),
		"count":      st.LastCount,
	})
}

// SyncImagesStatus GET /api/serverDeploy/images/sync/status
//
// 同步状态(给前端轮询/展示用)
func SyncImagesStatus(c *gin.Context) {
	handler.Success(c, service.GetSyncStatus())
}

// translateSyncErr 业务码翻译
func translateSyncErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAliyunImageNotConfig):
		handler.Error(c, CodeAliyunECSNotConfig, "阿里云 ECS 未配置,请检查 config.yaml")
	default:
		handler.Error(c, handler.CodeUnknown, "同步失败: "+err.Error())
	}
}
