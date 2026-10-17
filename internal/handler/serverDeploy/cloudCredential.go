// Package serverDeploy cloudCredential.go - 云平台凭证 HTTP 入口(2026-10-17 新增)
//
// 路由(在 router/serverDeploy.go 注册):
//
//	GET    /api/serverDeploy/credentials?type=&keyword=     列表(secret_key 自动 mask)
//	GET    /api/serverDeploy/credentials/:id               详情(secret_key 明文,仅这个入口返回)
//	POST   /api/serverDeploy/credentials                   新增
//	PUT    /api/serverDeploy/credentials/:id               编辑
//	DELETE /api/serverDeploy/credentials/:id               删除(软删)
//
// 业务码:
//
//	9010 CloudCredentialNotFound   凭证不存在
//	9011 CloudCredentialNameExists 同名同类型已存在
package serverDeploy

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"go_server/internal/handler"
	"go_server/internal/model"
	"go_server/internal/service"
)

const (
	CodeCloudCredentialNotFound   = 9010
	CodeCloudCredentialNameExists = 9011
)

// ListCloudCredentials GET /api/serverDeploy/credentials
//
// 列表返参里 secret_key 已经被 MaskCredentialList 替换成 mask 后的字符串
func ListCloudCredentials(c *gin.Context) {
	credType := c.Query("type")
	keyword := c.Query("keyword")
	list, total, err := service.ListCloudCredentials(credType, keyword)
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询凭证失败: "+err.Error())
		return
	}
	handler.Success(c, gin.H{
		"list":  service.MaskCredentialList(list),
		"total": total,
	})
}

// GetCloudCredential GET /api/serverDeploy/credentials/:id
//
// 详情返回明文 secret_key(给编辑表单预填用)
func GetCloudCredential(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cred, err := service.GetCloudCredential(id)
	if err != nil {
		if errors.Is(err, service.ErrCloudCredentialNotFound) {
			handler.Error(c, CodeCloudCredentialNotFound, "凭证不存在")
			return
		}
		handler.Error(c, handler.CodeUnknown, "查询失败: "+err.Error())
		return
	}
	handler.Success(c, cred)
}

// CreateCloudCredential POST /api/serverDeploy/credentials
func CreateCloudCredential(c *gin.Context) {
	var in model.CloudCredential
	if err := c.ShouldBindJSON(&in); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误: "+err.Error())
		return
	}
	// 入参校验(避免空字符串入库,跟 NOT NULL 列兼容)
	if in.Name == "" || in.Type == "" || in.AccessKey == "" || in.SecretKey == "" {
		handler.Error(c, handler.CodeParamsInvalid, "name / type / access_key / secret_key 都不能为空")
		return
	}
	if in.Type != model.CloudCredentialTypeAliyun && in.Type != model.CloudCredentialTypeTencent {
		handler.Error(c, handler.CodeParamsInvalid, "type 必须是 aliyun / tencent")
		return
	}
	if err := service.CreateCloudCredential(&in); err != nil {
		if errors.Is(err, service.ErrCloudCredentialNameExists) {
			handler.Error(c, CodeCloudCredentialNameExists, "同名同类型凭证已存在")
			return
		}
		handler.Error(c, handler.CodeUnknown, "创建失败: "+err.Error())
		return
	}
	handler.Success(c, gin.H{"id": in.ID})
}

// UpdateCloudCredential PUT /api/serverDeploy/credentials/:id
func UpdateCloudCredential(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var in model.CloudCredential
	if err := c.ShouldBindJSON(&in); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误: "+err.Error())
		return
	}
	if in.Name == "" || in.Type == "" || in.AccessKey == "" || in.SecretKey == "" {
		handler.Error(c, handler.CodeParamsInvalid, "name / type / access_key / secret_key 都不能为空")
		return
	}
	if in.Type != model.CloudCredentialTypeAliyun && in.Type != model.CloudCredentialTypeTencent {
		handler.Error(c, handler.CodeParamsInvalid, "type 必须是 aliyun / tencent")
		return
	}
	if err := service.UpdateCloudCredential(id, &in); err != nil {
		switch {
		case errors.Is(err, service.ErrCloudCredentialNotFound):
			handler.Error(c, CodeCloudCredentialNotFound, "凭证不存在")
		case errors.Is(err, service.ErrCloudCredentialNameExists):
			handler.Error(c, CodeCloudCredentialNameExists, "同名同类型凭证已存在")
		default:
			handler.Error(c, handler.CodeUnknown, "更新失败: "+err.Error())
		}
		return
	}
	handler.Success(c, gin.H{"id": id})
}

// DeleteCloudCredential DELETE /api/serverDeploy/credentials/:id
func DeleteCloudCredential(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := service.DeleteCloudCredential(id); err != nil {
		if errors.Is(err, service.ErrCloudCredentialNotFound) {
			handler.Error(c, CodeCloudCredentialNotFound, "凭证不存在")
			return
		}
		handler.Error(c, handler.CodeUnknown, "删除失败: "+err.Error())
		return
	}
	handler.Success(c, gin.H{"id": id})
}
