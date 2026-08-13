// Package main 是 go_server 服务的启动入口
//
// 启动流程:
//  1. 初始化日志(logger)
//  2. 加载配置(config + env 覆盖)
//  3. 初始化数据库(GORM + MySQL)
//  4. 初始化 Redis(token 缓存用)
//  5. 注册路由(router)
//  6. 启动 HTTP server
//
// 注意:本文件**不要添加业务代码**。业务代码应该在 internal/service 里。
package main

import (
	"fmt"

	"go.uber.org/zap"

	"go_server/config"
	"go_server/internal/model"
	"go_server/internal/router"
	"go_server/internal/service"
	"go_server/pkg/logger"
)

func main() {
	// 1. 日志最先初始化,后续模块都能用 logger.L
	logger.Init()
	defer logger.Sync()

	// 2. 初始化配置(yaml + env 覆盖)
	//    失败直接 Fatal 退出(没配置跑不起来)
	if err := config.InitConfig("config/config.yaml"); err != nil {
		logger.L.Fatal("配置文件加载失败", zap.Error(err))
	}

	// 3. 初始化数据库
	//    DB 失败只打 error,继续跑(可能是开发环境暂时没 DB)
	if err := model.InitDB(); err != nil {
		logger.L.Error("数据库连接失败", zap.Error(err))
		logger.L.Info("提示: 请确保 MySQL 已启动并配置正确的数据库")
	} else {
		logger.L.Info("数据库连接成功")
		// 保留原行为:即使 AutoMigrate 失败也不退出(原代码忽略错误)
		// 生产环境应该改用 migration 工具(详见 DEVELOPING.md)
		models := []interface{}{
			&model.AdminRoles{},
			&model.AdminMenus{},
			&model.AdminMenuOperations{},
			&model.AdminRoleMenus{},
			&model.AdminRoleOperations{},
			&model.AdminUsers{},
			&model.AdminDepartments{},
			&model.CodeEndpoints{},
			&model.BusinessProjects{},
			&model.BusinessProjectEndpoints{},
			&model.CodePackages{},
		}
		for _, m := range models {
			if err := model.DB.AutoMigrate(m); err != nil {
				logger.L.Error("AutoMigrate 失败", zap.Error(err))
			}
		}
	}

	// 4. 初始化 Redis(token 缓存用)
	if err := model.InitRedis(); err != nil {
		logger.L.Error("Redis 连接失败", zap.Error(err))
	} else {
		logger.L.Info("Redis 连接成功")
	}

	// 5. 注册所有路由(中间件 + handler)
	r := router.SetupRouter(config.AppConfig.Server.Mode)

	// 5.5 同步 gin 路由到 admin_menu_operations(加新接口不用手 INSERT 了)
	//     超管角色 ID 默认 1(见 DEVELOPING.md)
	if model.DB != nil {
		if _, err := service.SyncRoutes(r, 1); err != nil {
			logger.L.Error("SyncRoutes 失败", zap.Error(err))
		}
	}

	// 6. 启动 HTTP server(阻塞,直到 r.Run 返回错误)
	addr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	logger.L.Info("服务器启动成功",
		zap.String("url", "http://localhost"+addr),
		zap.Int("port", config.AppConfig.Server.Port),
	)
	if err := r.Run(addr); err != nil {
		logger.L.Fatal("服务器启动失败", zap.Error(err))
	}
}
