# go_server

基于 **Gin + GORM + Redis** 的后台管理服务,提供账号、角色、菜单、权限等基础管理能力。

---

## 技术栈

| 类别 | 选型 |
|------|------|
| 语言 | Go 1.25 |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) |
| ORM | [GORM](https://gorm.io) + MySQL |
| 缓存 / Token 存储 | [go-redis](https://github.com/redis/go-redis) |
| 配置 | YAML |
| 文件存储 | 阿里云 OSS |

---

## 目录结构

```
go_server/
├── cmd/                       # 可执行入口
│   └── server/                # 主程序入口
├── config/                    # 配置
│   ├── config.go              # 配置加载
│   ├── config.example.yaml    # 示例配置(进库)
│   └── config.yaml            # 本地真实配置(.gitignore 屏蔽)
├── internal/                  # 内部包(不对外暴露)
│   ├── handler/               # HTTP 业务处理
│   │   ├── auth.go            # 登录 / 注销 / 当前用户
│   │   ├── response.go        # 统一响应
│   │   ├── upload.go          # 文件上传
│   │   └── system/            # 系统管理(用户/角色/菜单)
│   ├── middleware/            # 中间件
│   │   ├── auth.go            # 鉴权
│   │   ├── cors.go            # 跨域
│   │   └── permission.go      # 权限
│   ├── model/                 # 数据模型 + DB 初始化
│   ├── router/                # 路由注册
│   └── service/               # 业务逻辑层
├── pkg/                       # 公共包
│   ├── cache/                 # token 缓存
│   └── logger/                # zap 日志初始化
├── Dockerfile                 # 多阶段构建
├── docker-compose.yml         # app + mysql + redis 编排
├── Makefile
├── .gitignore
└── README.md
```

---

## 快速开始

### 1. 准备环境

- Go >= 1.25
- MySQL >= 5.7
- Redis >= 5.0

### 2. 拉取依赖

```bash
go mod tidy
```

### 3. 修改配置

编辑 `config/config.yaml`,填入 MySQL / Redis / 阿里云 OSS 的真实信息。

### 4. 启动服务

```bash
# 方式 1:直接 go run
go run .

# 方式 2:用 Makefile(需要先安装 make)
make run

# 方式 3:编译后运行
go build -o bin/go_server.exe .
./bin/go_server.exe
```

启动成功后控制台会打印:
```
数据库连接成功
Redis 连接成功
服务器启动成功: http://localhost:8080
```

---

## 常用命令(Makefile)

```bash
make help          # 查看所有命令
make build         # 编译主程序到 bin/
make run           # 运行主程序
make test          # 跑测试
make fmt           # 格式化代码
make vet           # 静态检查
make tidy          # go mod tidy
make clean         # 清理构建产物
```

---

## API 路由

所有接口都以 `/api` 为前缀。

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/api/login` | 登录 | 否 |
| POST | `/api/logout` | 注销 | 否 |
| GET  | `/api/user/info` | 当前用户信息 | 是 |
| POST | `/api/upload/image` | 上传图片 | 是 |
| *    | `/api/system/admin_users` | 用户管理 | 是 |
| *    | `/api/system/admin_roles` | 角色管理 | 是 |
| *    | `/api/system/admin_menus` | 菜单管理 | 是 |
| *    | `/api/system/role_menus`  | 角色-菜单关系 | 是 |

---

## 鉴权流程

1. 客户端 `POST /api/login` 提交用户名/密码
2. 服务端校验通过后,生成 UUID token,存进 Redis(Hash 结构)
3. 后续请求在 Header 里带 `Authorization: <token>`
4. 中间件从 Redis 取出 token 数据,写入 gin.Context
5. 注销时直接删 Redis key

---

## 后续规划

参见项目内的整改计划(P1 / P2 任务):

- 抽 `service` 层
- 收口全局 DB / Redis 变量
- 敏感信息挪到环境变量
- CORS 改白名单
- 加测试

---

## License

待定
