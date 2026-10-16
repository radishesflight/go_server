// Package service upload_service.go - 文件上传业务
//
// 业务流程(UploadImage):
//  1. 校验扩展名(jpg/jpeg/png/gif/webp)
//  2. 校验大小(≤ 5MB)
//  3. 拿全局 Uploader(S3 兼容,启动期由 cmd/server/main.go 注入)
//  4. 上传 object(uploads/<timestamp><ext>)
//  5. 拼可访问 URL(bucket_host 或 endpoint 拼)
//
// 业务错误:见 internal/handler/upload.go 的翻译表
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"go_server/pkg/storage"
)

// 上传相关业务错误
var (
	ErrUploadNoFile         = errors.New("请选择要上传的文件")
	ErrUploadUnsupportedExt = errors.New("不支持的图片格式")
	ErrUploadTooLarge       = errors.New("图片大小不能超过 5MB")
	ErrUploadOSSNotConfig   = errors.New("对象存储未配置")
	ErrUploadClient         = errors.New("对象存储客户端初始化失败")
	ErrUploadBucket         = errors.New("对象存储 Bucket 无效")
	ErrUploadRead           = errors.New("文件读取失败")
	ErrUploadPut            = errors.New("文件上传失败")
)

const maxUploadSize = 5 * 1024 * 1024

// allowedImageExts 允许上传的图片扩展名
var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// contentTypeByExt 用扩展名反推 MIME(给 S3 PutObject 用)
var contentTypeByExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// UploadService 文件上传业务
type UploadService struct{}

// NewUploadService 构造 UploadService
func NewUploadService() *UploadService { return &UploadService{} }

// UploadImage 上传图片到对象存储(S3 兼容)
// filename 原始文件名, size 文件大小, open 回调返回文件流(io.ReadCloser)
// 返回:可访问的 URL
func (s *UploadService) UploadImage(filename string, size int64, open func() (io.ReadCloser, error)) (string, error) {
	if filename == "" {
		return "", ErrUploadNoFile
	}

	ext := strings.ToLower(path.Ext(filename))
	if !allowedImageExts[ext] {
		return "", ErrUploadUnsupportedExt
	}

	if size > maxUploadSize {
		return "", ErrUploadTooLarge
	}

	uploader := Uploader()
	if uploader == nil {
		return "", ErrUploadOSSNotConfig
	}

	f, err := open()
	if err != nil {
		return "", ErrUploadRead
	}
	defer f.Close()

	objectKey := uploader.ObjectKey(fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext))
	url, err := uploader.Upload(context.Background(), objectKey, f, size, contentTypeByExt[ext])
	if err != nil {
		// storage 包的错误翻译成 upload 业务错误(给前端统一 3xxx 业务码)
		switch {
		case errors.Is(err, storage.ErrNotConfig):
			return "", ErrUploadOSSNotConfig
		case errors.Is(err, storage.ErrClientInit):
			return "", ErrUploadClient
		case errors.Is(err, storage.ErrBucketInvalid):
			return "", ErrUploadBucket
		case errors.Is(err, storage.ErrUploadRead):
			return "", ErrUploadRead
		default:
			return "", ErrUploadPut
		}
	}
	return url, nil
}

// DeleteFileFromStorage 从对象存储删一个 object(objectKey 允许带 prefix,自动剥)
func (s *UploadService) DeleteFileFromStorage(objectKey string) error {
	if objectKey == "" {
		return nil
	}
	uploader := Uploader()
	if uploader == nil {
		return ErrUploadOSSNotConfig
	}
	return uploader.Delete(context.Background(), objectKey)
}
