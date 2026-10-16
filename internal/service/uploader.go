// Package service - 全局共享的对象存储 Uploader 单例
//
// 由 cmd/server/main.go 启动时调 SetUploader 注入,
// 业务 service 在 Upload*/Delete*/GetDownloadURL 里调 Uploader() 拿当前实例。
//
// 设计原因:uploader 持有 *minio.Client(goroutine-safe)+ bucket 等元信息,
// 每个 service 各自 new 浪费且测试时不方便 mock。
package service

import (
	"go_server/pkg/storage"
)

var globalUploader storage.Uploader

// SetUploader 由 main.go 启动时调一次
func SetUploader(u storage.Uploader) {
	globalUploader = u
}

// Uploader 业务侧拿当前 uploader(可能为 nil,表示未配置)
func Uploader() storage.Uploader {
	return globalUploader
}
