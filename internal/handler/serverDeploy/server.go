// Package serverDeploy server.go - 云服务器列表 HTTP 入口(2026-10-17 新增)
//
// 路由(在 router/serverDeploy.go 注册,菜单 code=servers 自动绑 op):
//
//	GET    /api/serverDeploy/servers?credential_id=&vendor=&region=&keyword=
//	       列服务器(从本地 DB)
//	POST   /api/serverDeploy/servers/sync?credential_id=  触发同步(credential_id 缺省=全部 aliyun 凭证)
//	GET    /api/serverDeploy/servers/sync/status           查询同步状态(轮询)
//
// 业务码:
//
//	9020 CloudServerSyncFail          同步失败
//	9021 CloudServerNoCredentials    没有可同步的阿里云凭证
//	9022 CloudServerSyncUnsupported  该厂商暂不支持同步(目前仅阿里云)
package serverDeploy

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"go_server/internal/handler"
	"go_server/internal/service"
)

const (
	CodeCloudServerSyncFail        = 9020
	CodeCloudServerNoCredentials   = 9021
	CodeCloudServerSyncUnsupported = 9022
)

// ListCloudServers GET /api/serverDeploy/servers
//
// 支持按 credential_id / vendor / region / keyword 过滤,前端表格用
func ListCloudServers(c *gin.Context) {
	credID, _ := strconv.ParseUint(c.Query("credential_id"), 10, 64)
	vendor := c.Query("vendor")
	region := c.Query("region")
	keyword := c.Query("keyword")

	list, total, err := service.ListCloudServers(credID, vendor, region, keyword)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询服务器列表失败: "+err.Error())
		return
	}
	// 简单前端分页用 page/size(前端传不传都行,这里再切)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 50
	}
	start := (page - 1) * size
	end := start + size
	if start > len(list) {
		start = len(list)
	}
	if end > len(list) {
		end = len(list)
	}
	handler.Success(c, gin.H{
		"list":  list[start:end],
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// SyncCloudServers POST /api/serverDeploy/servers/sync?credential_id=N
//
// credential_id 缺省或 0 → 同步所有 aliyun 类型凭证
func SyncCloudServers(c *gin.Context) {
	credID, _ := strconv.ParseUint(c.Query("credential_id"), 10, 64)
	status, n, err := service.SyncCloudServers(credID)
	if err != nil {
		switch err.Error() {
		case "没有可同步的阿里云凭证":
			handler.Error(c, CodeCloudServerNoCredentials, err.Error())
		default:
			handler.Error(c, CodeCloudServerSyncFail, status+": "+err.Error())
		}
		return
	}
	// 即便 err == nil,返回同步状态(包含 last_error;前端 alert 友好)
	st := service.GetServerSyncStatus()
	handler.Success(c, gin.H{
		"message":       "同步完成",
		"count":         n,
		"credential_id": credID,
		"last_at":       st.LastAt,
		"last_count":    st.LastCount,
	})
}

// SyncCloudServersStatus GET /api/serverDeploy/servers/sync/status
func SyncCloudServersStatus(c *gin.Context) {
	handler.Success(c, service.GetServerSyncStatus())
}
