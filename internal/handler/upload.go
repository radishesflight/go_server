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
func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Error(c, CodeUploadInvalid, "请选择要上传的文件")
		return
	}

	url, err := uploadSvc.UploadImage(file.Filename, file.Size, func() (io.ReadCloser, error) {
		// file.Open() 返回 multipart.File,实现了 io.ReadCloser
		return file.Open()
	})
	if err != nil {
		// 错误消息与原 handler 保持一致
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

// DeleteFileFromOSS 从 OSS 删除文件(保持原导出,逻辑下沉到 service)
func DeleteFileFromOSS(objectKey string) error {
	return uploadSvc.DeleteFileFromOSS(objectKey)
}
