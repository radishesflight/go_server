// Package config 提供全局配置加载
//
// 使用流程:
//  1. cmd/server/main.go 调 InitConfig("config/config.yaml")
//  2. 业务代码用 config.AppConfig.Xxx.Yyy 读字段
//
// 配置优先级(从高到低):
//  1. 环境变量(env > yaml)
//  2. config.yaml(本地真实配置,.gitignore 屏蔽)
//  3. config.example.yaml(示例,进库)
//
// env 命名规则:yaml 字段全大写 + 下划线
//
//	server.port            → SERVER_PORT
//	database.password      → DATABASE_PASSWORD
//	storage.access_key     → STORAGE_ACCESS_KEY
//
// 加新字段的步骤:
//  1. 在对应 XxxConfig struct 加字段
//  2. 在 config.example.yaml 加示例
//  3. 在 applyEnvOverrides 加 env 覆盖逻辑
//  4. 业务代码用 config.AppConfig.Xxx.Yyy
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
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	CORS       CORSConfig       `yaml:"cors"`
	Storage    StorageConfig    `yaml:"storage"`
	Jumpserver JumpserverConfig `yaml:"jumpserver"`
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

// StorageConfig 对象存储(S3 兼容协议)
//
// 支持的 provider:
//   - aliyun   阿里云 OSS(使用 S3 兼容 endpoint,推荐)
//   - tencent  腾讯云 COS(使用 S3 兼容 endpoint,推荐)
//   - minio    自建 MinIO / 其他 S3 兼容服务
//
// 阿里云 OSS S3 endpoint 形如:oss-cn-chengdu.aliyuncs.com
//
//	使用 path-style=false,Region 必填
//
// 腾讯云 COS S3 endpoint 形如:cos.ap-guangzhou.myqcloud.com
//
//	使用 path-style=false,Region 必填
//
// 业务侧调用方:pkg/storage.Uploader.Upload() / Delete() / PresignGet()
type StorageConfig struct {
	Provider   string `yaml:"provider"`    // aliyun / tencent / minio
	Endpoint   string `yaml:"endpoint"`    // S3 endpoint(不带 https://)
	Region     string `yaml:"region"`      // 地域(签名用,空则用 endpoint 第一段)
	AccessKey  string `yaml:"access_key"`  // 访问密钥
	SecretKey  string `yaml:"secret_key"`  // 私密密钥
	Bucket     string `yaml:"bucket"`      // bucket 名
	UseSSL     bool   `yaml:"use_ssl"`     // 是否 https(默认 true)
	PathStyle  bool   `yaml:"path_style"`  // true=path-style, false=virtual-host(默认)
	BucketHost string `yaml:"bucket_host"` // 自定义域名/cdn(可空,空则拼 endpoint)
	Prefix     string `yaml:"prefix"`      // 路径前缀(可空,如 "prod/")
}

// JumpserverConfig JumpServer Core API 配置
// 用于代码部署模块从 JMS 同步业务项目节点树
// 凭证可通过环境变量覆盖:JUMPSERVER_BASE_URL / JUMPSERVER_TOKEN / JUMPSERVER_ORG_ID
type JumpserverConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
	OrgID   string `yaml:"org_id"`
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

	// storage (S3 兼容对象存储)
	if v := os.Getenv("STORAGE_PROVIDER"); v != "" {
		c.Storage.Provider = v
	}
	if v := os.Getenv("STORAGE_ENDPOINT"); v != "" {
		c.Storage.Endpoint = v
	}
	if v := os.Getenv("STORAGE_REGION"); v != "" {
		c.Storage.Region = v
	}
	if v := os.Getenv("STORAGE_ACCESS_KEY"); v != "" {
		c.Storage.AccessKey = v
	}
	if v := os.Getenv("STORAGE_SECRET_KEY"); v != "" {
		c.Storage.SecretKey = v
	}
	if v := os.Getenv("STORAGE_BUCKET"); v != "" {
		c.Storage.Bucket = v
	}
	if v := os.Getenv("STORAGE_BUCKET_HOST"); v != "" {
		c.Storage.BucketHost = v
	}
	if v := os.Getenv("STORAGE_PREFIX"); v != "" {
		c.Storage.Prefix = v
	}
	if v := os.Getenv("STORAGE_USE_SSL"); v != "" {
		c.Storage.UseSSL = v == "true" || v == "1"
	}
	if v := os.Getenv("STORAGE_PATH_STYLE"); v != "" {
		c.Storage.PathStyle = v == "true" || v == "1"
	}

	// jumpserver
	if v := os.Getenv("JUMPSERVER_BASE_URL"); v != "" {
		c.Jumpserver.BaseURL = v
	}
	if v := os.Getenv("JUMPSERVER_TOKEN"); v != "" {
		c.Jumpserver.Token = v
	}
	if v := os.Getenv("JUMPSERVER_ORG_ID"); v != "" {
		c.Jumpserver.OrgID = v
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
