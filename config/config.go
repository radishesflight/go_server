package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig 全局配置(由 InitConfig 初始化)
var AppConfig *Config

// Config 顶层配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	CORS     CORSConfig     `yaml:"cors"`
	Aliyun   AliyunConfig   `yaml:"aliyun"`
}

// ServerConfig HTTP 服务
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// DatabaseConfig MySQL
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Dbname   string `yaml:"dbname"`
	Charset  string `yaml:"charset"`
	Prefix   string `yaml:"prefix"`
}

// RedisConfig Redis
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
}

// CORSConfig 跨域配置
// AllowedOrigins 支持 "*" 或以英文逗号分隔的 origin 列表
// 例:"http://localhost:5173,https://admin.example.com"
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

// AliyunConfig 阿里云
type AliyunConfig struct {
	OSS OSSConfig `yaml:"oss"`
}

// OSSConfig OSS 配置
type OSSConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	BucketName      string `yaml:"bucket_name"`
	BucketDomain    string `yaml:"bucket_domain"`
}

// InitConfig 从 yaml 加载配置,然后用环境变量覆盖
// 优先级:env > yaml
func InitConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	AppConfig = &Config{}
	if err := yaml.Unmarshal(data, AppConfig); err != nil {
		return err
	}

	// 环境变量覆盖(在 yaml 之后,空值不覆盖)
	applyEnvOverrides(AppConfig)

	// CORS 默认值
	if AppConfig.CORS.AllowedOrigins == "" {
		AppConfig.CORS.AllowedOrigins = "*"
	}
	return nil
}

// applyEnvOverrides 用环境变量覆盖 yaml 中的字段
// 任何对应环境变量被设置且非空时,优先使用环境变量
func applyEnvOverrides(c *Config) {
	// server
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Server.Port = n
		}
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
		c.Server.Mode = v
	}

	// database
	if v := os.Getenv("DATABASE_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DATABASE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Database.Port = n
		}
	}
	if v := os.Getenv("DATABASE_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DATABASE_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DATABASE_DBNAME"); v != "" {
		c.Database.Dbname = v
	}
	if v := os.Getenv("DATABASE_CHARSET"); v != "" {
		c.Database.Charset = v
	}
	if v := os.Getenv("DATABASE_PREFIX"); v != "" {
		c.Database.Prefix = v
	}

	// redis
	if v := os.Getenv("REDIS_HOST"); v != "" {
		c.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Redis.Port = n
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Redis.Db = n
		}
	}

	// cors
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		c.CORS.AllowedOrigins = v
	}

	// aliyun oss
	if v := os.Getenv("ALIYUN_OSS_ENDPOINT"); v != "" {
		c.Aliyun.OSS.Endpoint = v
	}
	if v := os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID"); v != "" {
		c.Aliyun.OSS.AccessKeyID = v
	}
	if v := os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET"); v != "" {
		c.Aliyun.OSS.AccessKeySecret = v
	}
	if v := os.Getenv("ALIYUN_OSS_BUCKET_NAME"); v != "" {
		c.Aliyun.OSS.BucketName = v
	}
	if v := os.Getenv("ALIYUN_OSS_BUCKET_DOMAIN"); v != "" {
		c.Aliyun.OSS.BucketDomain = v
	}
}

// SplitOrigins 把逗号分隔的 origin 字符串切成 slice
// "*" 保留为单元素 slice
func (c CORSConfig) SplitOrigins() []string {
	v := strings.TrimSpace(c.AllowedOrigins)
	if v == "" || v == "*" {
		return []string{"*"}
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
