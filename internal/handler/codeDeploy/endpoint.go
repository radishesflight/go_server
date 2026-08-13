// Package codeDeploy endpoint.go - 部署端字典 HTTP 入口
//
// 接口列表:
//
//	GET /api/codeDeploy/endpoints   查所有启用的端(下拉用)
package codeDeploy

import (
	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

var endpointSvc = service.NewEndpointService()

// ListEndpoints 查所有启用的端
// GET /api/codeDeploy/endpoints
func ListEndpoints(c *gin.Context) {
	list, err := endpointSvc.ListAll()
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询端列表失败")
		return
	}
	handler.Success(c, gin.H{"list": list})
}
