package main

import (
	"fmt"

	"go.uber.org/zap"

	"go_server/config"
	"go_server/internal/model"
	"go_server/internal/router"
	"go_server/pkg/logger"
)

func main() {
	// 日志最先初始化,后续模块都能用
	logger.Init()
	defer logger.Sync()

	// 初始化配置
	if err := config.InitConfig("config/config.yaml"); err != nil {
		logger.L.Fatal("配置文件加载失败", zap.Error(err))
	}

	// 初始化数据库
	if err := model.InitDB(); err != nil {
		logger.L.Error("数据库连接失败", zap.Error(err))
		logger.L.Info("提示: 请确保 MySQL 已启动并配置正确的数据库")
	} else {
		logger.L.Info("数据库连接成功")
		// 保留原行为:即使 AutoMigrate 失败也不退出(原代码忽略错误)
		if err := model.DB.AutoMigrate(&model.AdminRoles{}); err != nil {
			logger.L.Error("AutoMigrate AdminRoles 失败", zap.Error(err))
		}
		if err := model.DB.AutoMigrate(&model.AdminMenus{}); err != nil {
			logger.L.Error("AutoMigrate AdminMenus 失败", zap.Error(err))
		}
	}

	// 初始化 Redis
	if err := model.InitRedis(); err != nil {
		logger.L.Error("Redis 连接失败", zap.Error(err))
	} else {
		logger.L.Info("Redis 连接成功")
	}

	// 设置路由
	r := router.SetupRouter(config.AppConfig.Server.Mode)

	// 启动服务
	addr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	logger.L.Info("服务器启动成功",
		zap.String("url", "http://localhost"+addr),
		zap.Int("port", config.AppConfig.Server.Port),
	)
	if err := r.Run(addr); err != nil {
		logger.L.Fatal("服务器启动失败", zap.Error(err))
	}
}
