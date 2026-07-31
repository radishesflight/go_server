// Package logger 提供基于 zap 的统一日志初始化与全局访问
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// L 全局 logger,Init 之后方可使用
var L *zap.Logger

// Init 初始化全局 logger
// level:debug / info / warn / error,通过 ENV LOGGER_LEVEL 控制,默认 info
// development:通过 ENV LOGGER_DEV 控制(true/false),默认 false
func Init() {
	level := zapcore.InfoLevel
	if v := os.Getenv("LOGGER_LEVEL"); v != "" {
		_ = level.UnmarshalText([]byte(v))
	}

	dev := os.Getenv("LOGGER_DEV") == "true"

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewJSONEncoder(encCfg)

	if dev {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	L = zap.New(core, zap.AddCaller())
}

// Sync 刷盘(进程退出前调用)
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
