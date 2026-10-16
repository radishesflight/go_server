// Package storage - S3 兼容对象存储抽象
//
// 底层用 minio-go SDK,通过 S3 v4 签名协议同时支持:
//   - 阿里云 OSS(用 S3 兼容 endpoint)
//   - 腾讯云 COS(用 S3 兼容 endpoint)
//   - 自建 MinIO / 其他 S3 兼容服务
//
// 业务侧统一调 Upload() / Delete() / PresignGet(),不关心 provider。
//
// Provider 选择:由 config.AppConfig.Storage.Provider 决定
//   - "aliyun"  阿里云 OSS(S3 兼容)
//   - "tencent" 腾讯云 COS(S3 兼容)
//   - "minio"   自建 MinIO / 通用 S3
//   - 空 / 未知  → 默认按通用 S3 处理
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"go_server/config"
)

// 业务错误(给 handler 层翻译业务码)
var (
	ErrNotConfig     = errors.New("对象存储未配置")
	ErrClientInit    = errors.New("对象存储客户端初始化失败")
	ErrBucketInvalid = errors.New("对象存储 Bucket 无效")
	ErrUploadPut     = errors.New("对象存储上传失败")
	ErrUploadRead    = errors.New("对象存储文件读取失败")
	ErrObjectDelete  = errors.New("对象存储删除失败")
	ErrPresign       = errors.New("对象存储签名链接生成失败")
)

// Uploader 对象存储上传器接口
type Uploader interface {
	// Upload 把 data 推到 objectKey,返回可访问的 URL
	Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (string, error)
	// Delete 删一个 object(objectKey 允许带或不带 prefix)
	Delete(ctx context.Context, objectKey string) error
	// PresignGet 生成 GET 签名 URL(expires 内有效)
	//   forceDownload=true 时带 Content-Disposition: attachment(浏览器直接下载)
	PresignGet(ctx context.Context, objectKey string, expires time.Duration, forceDownload bool) (string, error)
	// ObjectKey 拼接带 prefix 的完整 key(便于业务侧只传相对路径)
	ObjectKey(relativePath string) string
	// StripPrefix 把存储里的全 key 还原成相对路径(给业务侧存 DB)
	StripPrefix(fullKey string) string
	// PublicURL 拼可访问的 URL(不签名,直接用 bucket_host 或 endpoint)
	PublicURL(objectKey string) string
}

// uploader 内部实现
type uploader struct {
	client *minio.Client
	cfg    config.StorageConfig
}

// New 根据 config 创建 Uploader
//
// 启动时调一次。后续并发安全(minio.Client 自身是 goroutine-safe)
func New(cfg config.StorageConfig) (Uploader, error) {
	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return nil, ErrNotConfig
	}

	// region:空时从 endpoint 推断(endpoint 形如 oss-cn-chengdu.aliyuncs.com 或 cos.ap-guangzhou.myqcloud.com)
	region := cfg.Region
	if region == "" {
		region = inferRegion(cfg.Endpoint)
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClientInit, err)
	}

	// 探测 bucket 是否存在(避免后端启动后第一次上传才发现配错)
	exists, err := client.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClientInit, err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: bucket %q 不存在", ErrBucketInvalid, cfg.Bucket)
	}

	return &uploader{client: client, cfg: cfg}, nil
}

// inferRegion 从 endpoint 推断 region
// 例:oss-cn-chengdu.aliyuncs.com → "oss-cn-chengdu" / cos.ap-guangzhou.myqcloud.com → "cos.ap-guangzhou"
//   注意:MinIO 自建通常是单个 region(可省略,这里 fallback 到 "us-east-1")
func inferRegion(endpoint string) string {
	host := endpoint
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "localhost" || host == "127.0.0.1" {
		return "us-east-1"
	}
	parts := strings.SplitN(host, ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "us-east-1"
}

// ObjectKey 拼带 prefix 的全 key
func (u *uploader) ObjectKey(relativePath string) string {
	relativePath = strings.TrimLeft(relativePath, "/")
	if u.cfg.Prefix == "" {
		return relativePath
	}
	prefix := strings.TrimRight(u.cfg.Prefix, "/")
	return prefix + "/" + relativePath
}

// StripPrefix 从全 key 拿回相对路径
func (u *uploader) StripPrefix(fullKey string) string {
	if u.cfg.Prefix == "" {
		return fullKey
	}
	prefix := strings.TrimRight(u.cfg.Prefix, "/") + "/"
	return strings.TrimPrefix(fullKey, prefix)
}

// PublicURL 拼可访问 URL(不签名,适合公共读 bucket)
func (u *uploader) PublicURL(objectKey string) string {
	if u.cfg.BucketHost != "" {
		return strings.TrimRight(u.cfg.BucketHost, "/") + "/" + objectKey
	}
	scheme := "https"
	if !u.cfg.UseSSL {
		scheme = "http"
	}
	host := u.cfg.Endpoint
	if u.cfg.PathStyle {
		return fmt.Sprintf("%s://%s/%s/%s", scheme, host, u.cfg.Bucket, objectKey)
	}
	// virtual-host style(默认): bucket.endpoint/objectKey
	// 阿里云 OSS / 腾讯云 COS 都是 virtual-host
	return fmt.Sprintf("%s://%s.%s/%s", scheme, u.cfg.Bucket, host, objectKey)
}

func (u *uploader) Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := u.client.PutObject(ctx, u.cfg.Bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUploadPut, err)
	}
	return u.PublicURL(objectKey), nil
}

func (u *uploader) Delete(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return nil
	}
	if err := u.client.RemoveObject(ctx, u.cfg.Bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("%w: %v", ErrObjectDelete, err)
	}
	return nil
}

func (u *uploader) PresignGet(ctx context.Context, objectKey string, expires time.Duration, forceDownload bool) (string, error) {
	reqParams := url.Values{}
	if forceDownload {
		// 让浏览器触发下载而不是预览
		// 注意:PresignGet 不直接支持 response-content-disposition,用 reqParams 注入
		reqParams.Set("response-content-disposition", `attachment`)
	}
	signed, err := u.client.PresignedGetObject(ctx, u.cfg.Bucket, objectKey, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPresign, err)
	}
	return signed.String(), nil
}
