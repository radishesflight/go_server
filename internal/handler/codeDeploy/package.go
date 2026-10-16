// Package codeDeploy package.go - 代码包 HTTP 入口
//
// 接口列表:
//
//	GET  /api/codeDeploy/packages?project_id=&endpoint_id=   列表
//	POST /api/codeDeploy/packages                            上传(multipart/form-data)
//	GET  /api/codeDeploy/packages/:id                        单条
//	POST /api/codeDeploy/packages/:id/pull                   触发部署(目前 mock)
//
// 业务码翻译:
//
//	service.ErrEndpointNotFound     → CodeEndpointNotFound
//	service.ErrProjectNotFound      → CodeProjectNotFound
//	service.ErrProjectNoEndpoint    → CodeProjectNoEndpoint
//	service.ErrPackageNotFound      → CodePackageNotFound
//	service.ErrPackageExtInvalid    → CodePackageExtInvalid
//	service.ErrPackageTooLarge      → CodePackageTooLarge
//	service.ErrUpload*              → CodeUpload* (3xxx)
package codeDeploy

import (
	"errors"
	"io"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

var packageSvc = service.NewCodePackageService()

// ListPackages 列表
// GET /api/codeDeploy/packages?project_id=&asset_id=&asset_path_id=
// 2026-10-16 改造:加 asset_id / asset_path_id 过滤,删 endpoint_id
func ListPackages(c *gin.Context) {
	projectID, _ := strconv.ParseUint(c.Query("project_id"), 10, 64)
	assetID, _ := strconv.ParseUint(c.Query("asset_id"), 10, 64)
	assetPathID, _ := strconv.ParseUint(c.Query("asset_path_id"), 10, 64)
	list, err := packageSvc.List(uint(projectID), uint(assetID), uint(assetPathID))
	if err != nil {
		handler.Error(c, handler.CodeUnknown, "查询代码包失败")
		return
	}
	handler.Success(c, gin.H{"list": list})
}

// GetPackage 单条
// GET /api/codeDeploy/packages/:id
func GetPackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的代码包ID")
		return
	}
	p, err := packageSvc.Get(uint(id))
	if err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			handler.Error(c, handler.CodePackageNotFound, "代码包不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "查询代码包失败")
		}
		return
	}
	handler.Success(c, p)
}

// UploadPackage 上传代码包
// POST /api/codeDeploy/packages
// multipart/form-data fields:
//   - file           必填,apk/zip/tar.gz
//   - asset_id       必填,目标资产 business_assets.id
//   - asset_path_id  可选,目标路径 business_asset_paths.id(不填=不指定 cwd)
//   - name           必填,前端填的原始名(不含版本号/扩展名)
//   - note           可选
//
// 服务端自动生成:build_time = 当前时间,uploader_id = token.user_id
func UploadPackage(c *gin.Context) {
	assetID, _ := strconv.ParseUint(c.PostForm("asset_id"), 10, 64)
	assetPathID, _ := strconv.ParseUint(c.PostForm("asset_path_id"), 10, 64)
	name := c.PostForm("name")
	note := c.PostForm("note")
	if assetID == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "请选择目标资产")
		return
	}
	if name == "" {
		handler.Error(c, handler.CodeParamsInvalid, "请填写包名")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		handler.Error(c, handler.CodeUploadInvalid, "请选择要上传的文件")
		return
	}

	// 当前登录用户 ID(从 auth middleware 注入的 c.Get("user_id") 拿)
	uploaderID := uint(0)
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(uint); ok2 {
			uploaderID = id
		}
	}

	pkg, err := packageSvc.Upload(
		name, file.Filename, file.Size,
		func() (io.ReadCloser, error) { return file.Open() },
		note, uploaderID,
		uint(assetID), uint(assetPathID),
	)
	if err != nil {
		translateUploadErr(c, err)
		return
	}
	handler.Success(c, pkg)
}

// translateUploadErr 翻译代码包上传/校验相关错误
func translateUploadErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPackageExtInvalid):
		handler.Error(c, handler.CodePackageExtInvalid, "文件扩展名与所选端不匹配")
	case errors.Is(err, service.ErrPackageTooLarge):
		handler.Error(c, handler.CodePackageTooLarge, "代码包不能超过 200MB")
	case errors.Is(err, service.ErrEndpointNotFound):
		handler.Error(c, handler.CodeEndpointNotFound, "端不存在")
	case errors.Is(err, service.ErrProjectNotFound):
		handler.Error(c, handler.CodeProjectNotFound, "项目不存在")
	case errors.Is(err, service.ErrProjectNoEndpoint):
		handler.Error(c, handler.CodeProjectNoEndpoint, "该项目未配置该端")
	case errors.Is(err, service.ErrPackageCreate):
		handler.Error(c, handler.CodeUnknown, "代码包入库失败")
	case errors.Is(err, service.ErrUploadOSSNotConfig):
		handler.Error(c, handler.CodeOSSNoConfig, "OSS 未配置")
	case errors.Is(err, service.ErrUploadClient):
		handler.Error(c, handler.CodeOSSClient, "OSS 客户端创建失败")
	case errors.Is(err, service.ErrUploadBucket):
		handler.Error(c, handler.CodeOSSBucket, "OSS Bucket 获取失败")
	case errors.Is(err, service.ErrUploadRead):
		handler.Error(c, handler.CodeUploadRead, "文件读取失败")
	case errors.Is(err, service.ErrUploadPut):
		handler.Error(c, handler.CodeUploadPut, "文件上传失败")
	default:
		handler.Error(c, handler.CodeUploadPut, "文件上传失败")
	}
}

// PullPackage 触发部署
// POST /api/codeDeploy/packages/:id/pull
// 真实逻辑:查 deploy_targets → 拿资产(jms_id) + 路径 → 生成 OSS 签名下载 URL →
//          拼 cd+curl+unzip 命令 → 调 JMS ad-hoc 任务 → 后端内部轮询到终态 → 一次性返回 stdout
func PullPackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的代码包ID")
		return
	}
	res, err := packageSvc.Pull(uint(id))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPackageNotFound):
			handler.Error(c, handler.CodePackageNotFound, "代码包不存在")
		case errors.Is(err, service.ErrPackageNoDeployTarget):
			handler.Error(c, handler.CodePackageNoDeployTarget, "代码包未配置部署目标")
		case errors.Is(err, service.ErrPackagePullFail):
			// 部署失败时 res 仍带 stdout 等详情,塞到 data 给前端展示
			handler.ErrorWithData(c, handler.CodePackagePullFail, err.Error(), res)
		default:
			handler.Error(c, handler.CodeUnknown, "部署失败")
		}
		return
	}
	handler.Success(c, res)
}

// DeletePackage 删除代码包(软删 DB + 删 OSS)
// DELETE /api/codeDeploy/packages/:id
func DeletePackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的代码包ID")
		return
	}
	if err := packageSvc.Delete(uint(id)); err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			handler.Error(c, handler.CodePackageNotFound, "代码包不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "删除代码包失败")
		}
		return
	}
	handler.Success(c, nil)
}

// GetDownloadURL 返回 OSS 签名下载 URL(带 Content-Disposition: attachment)
// GET /api/codeDeploy/packages/:id/download-url
// 前端拿到 URL 后用 <a download> 触发浏览器下载
func GetDownloadURL(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		handler.Error(c, handler.CodeParamsInvalid, "无效的代码包ID")
		return
	}
	url, err := packageSvc.GetDownloadURL(uint(id))
	if err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			handler.Error(c, handler.CodePackageNotFound, "代码包不存在")
		} else if errors.Is(err, service.ErrUploadOSSNotConfig) {
			handler.Error(c, handler.CodeOSSNoConfig, "OSS 未配置")
		} else {
			handler.Error(c, handler.CodeUnknown, "获取下载链接失败")
		}
		return
	}
	handler.Success(c, url)
}
