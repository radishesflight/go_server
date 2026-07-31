# go_server 开发手册

> 本手册面向**继续在本项目上开发**的工程师。
> 阅读完你应该能:加新接口 / 加新业务 / 调业务码 / 写测试 / 提交代码。

---

## 1. 项目结构

```
go_server/
├── cmd/
│   └── server/main.go          # 启动入口(勿改)
├── config/
│   ├── config.go                # 配置结构 + env 覆盖
│   ├── config.example.yaml      # 配置示例(进库)
│   └── config.yaml              # 真实配置(.gitignore 屏蔽)
├── internal/                    # 内部包(对外不可见)
│   ├── handler/                 # HTTP 入口(参数解析 + 调 service + 业务码翻译)
│   │   ├── bizcode.go          # 业务码常量
│   │   ├── response.go         # Success / Error 工具
│   │   ├── auth.go             # 登录 / 注销 / 当前用户
│   │   ├── upload.go           # 上传
│   │   └── system/             # 系统管理子包
│   │       ├── adminUsers.go   # 用户 CRUD
│   │       ├── adminRoles.go   # 角色 CRUD
│   │       ├── adminMenus.go   # 菜单 CRUD
│   │       └── roleMenu.go     # 角色-菜单/权限分配
│   ├── middleware/              # 中间件
│   │   ├── cors.go             # 跨域
│   │   ├── auth.go             # 鉴权
│   │   └── permission.go       # 权限
│   ├── model/                   # GORM 数据模型 + DB/RDB 初始化
│   ├── router/                  # 路由注册
│   │   ├── router.go           # 主路由
│   │   └── system/             # system 子包
│   └── service/                 # 业务逻辑(纯函数 + DB 操作)
└── pkg/                         # 公共包(可被外部引用)
    ├── cache/                   # token 缓存(Redis)
    └── logger/                  # zap 日志初始化
```

**关键原则**:
- `handler` 只做参数解析 + 调 `service` + **业务码翻译**
- `service` 是业务核心,**所有 DB 操作在这里**
- `model` 只放数据模型 + DB 初始化
- 全局 `model.DB` / `model.RDB` 仅在 `service` 内被引用

---

## 2. 业务码规范(⭐ 重要)

后端用**业务码模式**:HTTP 状态码恒为 200,真实状态在 `body.code` 里。

### 业务码定义
见 `internal/handler/bizcode.go`,分组规则:

| 段位 | 含义 | 例子 |
|------|------|------|
| 0 | 成功 | `CodeSuccess = 0` |
| 1xxx | 鉴权 / 账号 / 角色 | `CodeUserNotFound = 1004` |
| 2xxx | 权限 | `CodeNoPermission = 2001` |
| 3xxx | 文件 / OSS | `CodeUploadSize = 3003` |
| 4xxx | 参数 | `CodeParamsInvalid = 4001` |
| 9xxx | 通用 | `CodeUnknown = 9999` |

### 调用方式

```go
// 成功
handler.Success(c, data)
// 业务错误(必须传业务码)
handler.Error(c, handler.CodeUserNotFound, "用户不存在")
```

### 加新业务码的流程

1. 在 `internal/handler/bizcode.go` 加常量(选合适段位)
2. 在 `internal/service/<your_service>.go` 抛 `errors.New(...)`(或定义 `ErrXxx = errors.New(...)`)
3. 在 `internal/handler/<your_handler>.go` 用 `errors.Is` 翻译
4. 在前端 `src/constants/bizcode.js` 同步加常量

---

## 3. 加一个新接口(完整示例)

假设要加一个**订单管理**的 CRUD 接口。

### 步骤 1:定义 model
`internal/model/order.go`(新文件):

```go
package model

import (
    "go_server/config"

    "gorm.io/gorm"
)

// Order 订单
// 表名: <prefix>orders(默认 go_orders)
type Order struct {
    BaseModel
    UserID   uint    `gorm:"index;not null" json:"user_id"`
    Amount   float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
    Status   int     `gorm:"default:0" json:"status"`
}

func (Order) TableName() string {
    return config.AppConfig.Database.Prefix + "orders"
}
```

### 步骤 2:写 service(业务核心)
`internal/service/order_service.go`(新文件):

```go
package service

import (
    "errors"
    "go_server/internal/model"
)

// 业务错误(与 handler 翻译对应)
var (
    ErrOrderNotFound  = errors.New("订单不存在")
    ErrOrderCreate    = errors.New("创建订单失败")
    ErrOrderUpdate    = errors.New("更新订单失败")
    ErrOrderDelete    = errors.New("删除订单失败")
)

type OrderService struct{}

func NewOrderService() *OrderService { return &OrderService{} }

// GetList 分页查询
func (s *OrderService) GetList(page, size int) ([]map[string]interface{}, int64, error) {
    var orders []model.Order
    var total int64
    db := model.DB.Model(&model.Order{})
    db.Count(&total)
    db.Offset((page - 1) * size).Limit(size).Find(&orders)
    // 转 map 给前端(参考 admin_user_service.go)
    return nil, total, nil
}

// Get 单条
func (s *OrderService) Get(id int) (map[string]interface{}, error) {
    var order model.Order
    if err := model.DB.First(&order, id).Error; err != nil {
        return nil, ErrOrderNotFound
    }
    return nil, nil
}

// Create / Update / Delete 类似
```

### 步骤 3:写 handler(参数 + 翻译)
`internal/handler/system/order.go`(新文件,放 system 子包):

```go
package system

import (
    "errors"
    "strconv"
    "go_server/internal/handler"
    "go_server/internal/service"
    "github.com/gin-gonic/gin"
)

var orderSvc = service.NewOrderService()

func GetOrdersList(c *gin.Context) {
    page, _ := strconv.Atoi(c.Query("page"))
    size, _ := strconv.Atoi(c.Query("size"))
    list, total, _ := orderSvc.GetList(page, size)
    handler.Success(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

func DeleteOrder(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        handler.Error(c, handler.CodeParamsInvalid, "无效的订单ID")
        return
    }
    if err := orderSvc.Delete(id); err != nil {
        if errors.Is(err, service.ErrOrderNotFound) {
            handler.Error(c, handler.CodeOrderNotFound, "订单不存在")
        } else {
            handler.Error(c, handler.CodeUnknown, "删除订单失败")
        }
        return
    }
    handler.Success(c, nil)
}
```

### 步骤 4:加业务码
`internal/handler/bizcode.go` 加:

```go
// 6xxx 订单
const (
    CodeOrderNotFound = 6001
    CodeOrderDuplicate = 6002
)
```

### 步骤 5:注册路由
`internal/router/system/order.go`(新文件):

```go
package system

import (
    "go_server/internal/handler/system"
    "go_server/internal/middleware"
    "github.com/gin-gonic/gin"
)

func OrderRoutes(rg *gin.RouterGroup) {
    g := rg.Group("/system/orders")
    g.Use(middleware.AuthMiddleware(), middleware.PermissionMiddleware())
    {
        g.GET("/list", system.GetOrdersList)
        g.DELETE("/:id", system.DeleteOrder)
    }
}
```

`internal/router/router.go` 加一行:

```go
api := r.Group("/api")
{
    // ... 现有
    system.OrderRoutes(api)
}
```

### 步骤 6:同步前端
前端 `src/constants/bizcode.js` 加:

```js
OrderNotFound: 6001,
OrderDuplicate: 6002,
```

前端 `src/api/system/order/index.js` 加 API 封装。

---

## 4. 配置规范

### 配置文件优先级
**环境变量 > YAML 配置**

```
config.example.yaml       (示例,进库)
config.yaml               (本地真实配置,.gitignore 屏蔽)
环境变量                  (最高优先级,生产用)
```

### 加新配置字段的流程

1. `config/config.go` 在对应 `XxxConfig` struct 加字段
2. `config/config.example.yaml` 加示例
3. `config/config.go` 在 `applyEnvOverrides` 加覆盖逻辑
4. 在业务代码用 `config.AppConfig.Xxx.Yyy`

### env 命名规则

yaml 字段名 → 全大写 + 下划线

```yaml
database:
  password: "xxx"  →  DATABASE_PASSWORD=xxx
aliyun:
  oss:
    access_key_id: "xxx"  →  ALIYUN_OSS_ACCESS_KEY_ID=xxx
```

---

## 5. 测试规范

### 单元测试(`*_test.go`)

- 文件名:`<file>_test.go`,与被测文件同包
- 函数名:`TestXxx`
- 框架:标准库 `testing`

**示例**(`internal/service/menu_tree_test.go`):

```go
package service

import (
    "testing"
    "go_server/internal/model"
)

func TestBuildMenuTree_SingleRoot(t *testing.T) {
    menus := []model.AdminMenus{
        {BaseModel: model.BaseModel{ID: 1}, Name: "根菜单", ParentID: 0},
    }
    roots := BuildMenuTree(menus)
    if len(roots) != 1 {
        t.Fatalf("expected 1 root, got %d", len(roots))
    }
}
```

### 跑测试

```bash
go test ./...                  # 全部
go test -v ./internal/service  # 单包
go test -cover ./...           # 覆盖率
```

---

## 6. 日志规范

用 `pkg/logger/logger.go` 的全局 `logger.L`:

```go
import "go_server/pkg/logger"
import "go.uber.org/zap"

logger.L.Info("用户登录", zap.String("username", username), zap.Uint("user_id", uid))
logger.L.Error("数据库连接失败", zap.Error(err))
logger.L.Fatal("启动失败,退出", zap.Error(err))  // 进程退出
```

**别再用 `log.Println`**,统一用 zap。

---

## 7. 命名约定

| 类别 | 规则 | 例子 |
|------|------|------|
| 包名 | 小写单词,不复数 | `service`, `handler` |
| 文件名 | 驼峰,按业务命名 | `admin_user_service.go` |
| 公开函数 | 大驼峰 | `GetAdminUserList` |
| 私有函数 | 小驼峰 | `formatUser` |
| 常量 | 大驼峰 | `CodeUserNotFound` |
| 业务错误 | `ErrXxx = errors.New(...)` | `ErrUserNotFound` |
| 业务码 | `CodeXxx`,分组前缀 | `CodeUserNotFound = 1004` |
| SQL 表名 | `<prefix>xxx`,蛇形 | `go_admin_users` |
| GORM 模型 | 大驼峰,struct 名 | `AdminUsers` |
| 路由 URL | 蛇形,复数 | `/api/system/adminUsers` |

---

## 8. 提交规范(commit message)

```
<type>: <subject>

<body>(可选)

<footer>(可选)
```

**type 类别**:
- `feat` 新功能
- `fix` 修复 bug
- `refactor` 重构(不改功能)
- `docs` 文档
- `test` 测试
- `chore` 构建 / 配置

**示例**:
```
feat: 加订单管理 CRUD

- service: GetList/Get/Create/Update/Delete
- handler: system/order.go
- router: /api/system/orders
- 业务码: 6001-6009
```

---

## 9. 常见错误 FAQ

### Q: 加新接口后前端报 404?
A: 检查 3 处:
1. `internal/router/system/<your>.go` 是否注册
2. `internal/router/router.go` 是否调用 `<Your>Routes(api)`
3. HTTP method + path 是否拼对

### Q: 业务码返回 0 但前端没拿到 data?
A: 检查 `handler.Success(c, data)` 中 data 是否是 nil;
非 nil 但 JSON 是 `null` 是 `omitempty` 把空 struct / map 过滤了。

### Q: 启动报"找不到 yaml 文件"?
A: `config.InitConfig` 用相对路径 `config/config.yaml`,确认 cwd 在项目根目录。

### Q: AutoMigrate 不生效?
A: 在 `cmd/server/main.go` 初始化时调,加新表后要重启服务。

### Q: 想加新权限但没生效?
A: 1) `RolePermission` 表里加 `(role_id, permission_code)`
2) 用户的 token 缓存里可能缓存了旧权限,让用户重新登录
3) 或者后端 `assignMenusToRole` 接口分配

---

## 10. 调试技巧

### 启动时打开 debug 日志
```bash
GIN_MODE=debug go run ./cmd/server
```

### 看具体请求
```go
logger.L.Info("xxx", zap.Any("data", data), zap.Error(err))
```

### 真机调接口
```bash
# 登录
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"xxx"}'

# 拿 token 调接口
curl -H "Authorization: <token>" http://localhost:8080/api/user/info
```

### 容器化
```bash
docker compose up -d
docker compose logs -f app
```

---

**祝开发愉快 🎉 有问题先看 FAQ,再问同事。**
