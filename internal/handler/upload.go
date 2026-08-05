// Package handler upload.go - 文件上传 HTTP 入口
//
// 业务流程:
//  1. 从 multipart form 拿文件(c.FormFile)
//  2. 调 service.UploadImage 做校验 + 上传
//  3. 翻译 service 错误为业务码
//
// 支持的图片格式:.jpg .jpeg .png .gif .webp
// 大小限制:5MB(在 service 里 hard-coded)
//
// 业务码翻译表:
//
//	service.ErrUploadNoFile          → CodeUploadInvalid
//	service.ErrUploadUnsupportedExt  → CodeUploadType
//	service.ErrUploadTooLarge        → CodeUploadSize
//	service.ErrUploadOSSNotConfig    → CodeOSSNoConfig
//	service.ErrUploadClient          → CodeOSSClient
//	service.ErrUploadBucket          → CodeOSSBucket
//	service.ErrUploadRead            → CodeUploadRead
//	service.ErrUploadPut             → CodeUploadPut
package handler

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"

	"go_server/internal/service"
)

// uploadSvc 上传业务入口
var uploadSvc = service.NewUploadService()

// UploadImage 处理图片上传
// POST /api/upload/image (multipart/form-data, field name = "file")
// 成功: {code: 0, data: {url: "<oss-cdn-url>"}}
// 失败: {code: <业务码>, msg: <错误文案>}
func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Error(c, CodeUploadInvalid, "请选择要上传的文件")
		return
	}

	// 第三个参数是回调函数,service 内部调它拿文件流
	// (不能直接传 file.Open() 因为它返回 multipart.File,service 强类型用 io.ReadCloser)
	url, err := uploadSvc.UploadImage(file.Filename, file.Size, func() (io.ReadCloser, error) {
		// file.Open() 返回 multipart.File,实现了 io.ReadCloser
		return file.Open()
	})
	if err != nil {
		// 业务码翻译
		switch {
		case errors.Is(err, service.ErrUploadNoFile):
			Error(c, CodeUploadInvalid, "请选择要上传的文件")
		case errors.Is(err, service.ErrUploadUnsupportedExt):
			Error(c, CodeUploadType, "不支持的图片格式")
		case errors.Is(err, service.ErrUploadTooLarge):
			Error(c, CodeUploadSize, "图片大小不能超过 5MB")
		case errors.Is(err, service.ErrUploadOSSNotConfig):
			Error(c, CodeOSSNoConfig, "OSS 未配置")
		case errors.Is(err, service.ErrUploadClient):
			Error(c, CodeOSSClient, "OSS 客户端创建失败")
		case errors.Is(err, service.ErrUploadBucket):
			Error(c, CodeOSSBucket, "OSS Bucket 获取失败")
		case errors.Is(err, service.ErrUploadRead):
			Error(c, CodeUploadRead, "文件读取失败")
		case errors.Is(err, service.ErrUploadPut):
			Error(c, CodeUploadPut, "文件上传失败")
		default:
			Error(c, CodeUploadPut, "文件上传失败")
		}
		return
	}

	Success(c, gin.H{
		"url": url,
	})
}
