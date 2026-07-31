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

var (
	DB  *gorm.DB
	RDB *redis.Client
)

type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `gorm:"type:datetime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:datetime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

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
