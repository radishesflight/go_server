// Package model 提供 GORM 数据模型 + DB/RDB 初始化
//
// 全局变量:
//   DB  - gorm.DB(MySQL 连接)
//   RDB - redis.Client(Redis 连接)
//
// 重要:DB / RDB 是 package-level 变量,在 service 层直接使用。
// handler 层**不要**直接访问,必须通过 service。
//
// 加新 model 的流程:
//  1. 在本目录加新文件(例 order.go)
//  2. struct 嵌入 BaseModel(获取 ID/时间戳/软删除)
//  3. 实现 TableName() 方法(加 prefix)
//  4. 在 cmd/server/main.go 加 AutoMigrate(&Order{})
//  5. 业务用 service 包访问(不直接用 DB)
package model

import (
	"context"
	"fmt"
	"time"

	"go_server/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB 全局 gorm.DB(由 InitDB 初始化)
// 业务代码通过 service 层访问,不要直接引用
var DB *gorm.DB

// RDB 全局 redis.Client(由 InitRedis 初始化)
// token 缓存在 pkg/cache 包,内部用 RDB
var RDB *redis.Client

// BaseModel 基础字段,所有业务表都嵌入
//   ID        - 主键自增
//   CreatedAt - 创建时间
//   UpdatedAt - 更新时间
//   DeletedAt - 软删除时间(gorm.DeletedAt 类型,有专门的软删除语义)
//     gorm:"index" 会自动建索引
//     json:"-" 不暴露给前端(软删状态前端不需要)
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `gorm:"type:datetime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:datetime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// FormatTime 把 time.Time 格式化为前端友好的字符串
// 输出:"2006-01-02 15:04:05"
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// InitDB 初始化 MySQL 连接
// 由 cmd/server/main.go 启动时调
// 失败时 main 进程会继续跑(只 log error),开发环境可能暂时没 DB
func InitDB() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Host,
		config.AppConfig.Database.Port,
		config.AppConfig.Database.Dbname,
		config.AppConfig.Database.Charset,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	return nil
}

// InitRedis 初始化 Redis 连接 + 立即 ping 一次确认通
// 失败同上,只 log 不退出
func InitRedis() error {
	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.AppConfig.Redis.Host, config.AppConfig.Redis.Port),
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.Db,
	})

	ctx := context.Background()
	if err := RDB.Ping(ctx).Err(); err != nil {
		return err
	}

	return nil
}
