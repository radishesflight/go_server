package service

import (
	"errors"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"go_server/config"
)

// 上传相关业务错误
var (
	ErrUploadNoFile         = errors.New("请选择要上传的文件")
	ErrUploadUnsupportedExt = errors.New("不支持的图片格式")
	ErrUploadTooLarge       = errors.New("图片大小不能超过 5MB")
	ErrUploadOSSNotConfig   = errors.New("OSS 未配置")
	ErrUploadClient         = errors.New("OSS 客户端创建失败")
	ErrUploadBucket         = errors.New("OSS Bucket 获取失败")
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

// UploadService 文件上传业务
type UploadService struct{}

// NewUploadService 构造 UploadService
func NewUploadService() *UploadService { return &UploadService{} }

// UploadImage 上传图片到阿里云 OSS
// filename 原始文件名, size 文件大小, open 回调返回文件流(io.ReadCloser)
// 返回:可访问的 URL
func (s *UploadService) UploadImage(filename string, size int64, open func() (io.ReadCloser, error)) (string, error) {
	if filename == "" {
		return "", ErrUploadNoFile
	}

	ext := path.Ext(filename)
	if !allowedImageExts[ext] {
		return "", ErrUploadUnsupportedExt
	}

	if size > maxUploadSize {
		return "", ErrUploadTooLarge
	}

	cfg := config.AppConfig.Aliyun.OSS
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.BucketName == "" {
		return "", ErrUploadOSSNotConfig
	}

	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return "", ErrUploadClient
	}
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return "", ErrUploadBucket
	}

	f, err := open()
	if err != nil {
		return "", ErrUploadRead
	}
	defer f.Close()

	objectKey := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)
	if err := bucket.PutObject(objectKey, f); err != nil {
		return "", ErrUploadPut
	}

	url := cfg.BucketDomain
	if url == "" {
		url = fmt.Sprintf("https://%s.%s/%s", cfg.BucketName, cfg.Endpoint, objectKey)
	} else {
		url = url + "/" + objectKey
	}
	return url, nil
}

// DeleteFileFromOSS 从 OSS 删除文件
func (s *UploadService) DeleteFileFromOSS(objectKey string) error {
	if objectKey == "" {
		return nil
	}
	cfg := config.AppConfig.Aliyun.OSS
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.BucketName == "" {
		return errors.New("OSS 未配置")
	}
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(objectKey)
}
