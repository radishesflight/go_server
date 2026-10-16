// Package service route_sync.go - 启动时同步 gin 路由到 admin_menu_operations
//
// 目的:解决"加新接口忘了 INSERT 到 admin_menu_operations"的事故
//
// 工作流:
//  1. cmd/server/main.go 启动时,在 SetupRouter 之后、HTTP serve 之前调 SyncRoutes
//  2. SyncRoutes 拿 gin.Engine.Routes() 拿到所有已注册路由
//  3. 过滤出 /api/system/* 下、需要 PermissionMiddleware 的路由
//  4. 对每条 (method, path):
//     - 不存在 → 解析 path 第一段(/api/system/<code>/...)反查 admin_menus.code
//     - 找到 menu_id → INSERT(默认 name=method+path)
//  5. 有新增 → 给 roleID=1(超管)BumpVersion,让 admin token 懒重载拿到新权限
//
// 注意:
//   - 不会重复 INSERT(唯一索引 idx_op_method_path 保证)
//   - 不会覆盖已有(name 已有就保留)
//   - 不会同步公开路由(/api/login, /api/user/info 等)
package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go_server/internal/model"
	"go_server/pkg/cache"
	"go_server/pkg/logger"
)

// SyncRoutesResult 同步结果(给启动日志用)
type SyncRoutesResult struct {
	Total      int      // gin 里注册的路由数(过滤后)
	Existing   int      // 已存在于 DB 的
	Inserted   int      // 新插入的
	InsertList []string // 新插入的 (method path) 列表
	Skipped    int      // 跳过的(找不到对应 menu.code 的)
	SkipList   []string
}

// SyncRoutes 扫 gin 路由,补齐 admin_menu_operations
// r: SetupRouter 返回的 gin.Engine
// adminRoleID: 超管角色 ID(默认 1,新插入的 operation 自动给这个角色关联,不然没人能用)
func SyncRoutes(r *gin.Engine, adminRoleID uint) (*SyncRoutesResult, error) {
	res := &SyncRoutesResult{}

	// 1. 拿到所有路由
	allRoutes := r.Routes()

	// 2. 过滤出 /api/system/* 下的(只有这些是 PermissionMiddleware 管的)
	filtered := make([]gin.RouteInfo, 0, len(allRoutes))
	for _, route := range allRoutes {
		// 只同步 PermissionMiddleware 管理的两类前缀
		if strings.HasPrefix(route.Path, "/api/system/") ||
			strings.HasPrefix(route.Path, "/api/codeDeploy/") {
			filtered = append(filtered, route)
		}
	}
	res.Total = len(filtered)
	if res.Total == 0 {
		return res, nil
	}

	// 3. 一次性加载所有 menu(用 code 索引)
	menuByCode := make(map[string]model.AdminMenus)
	var menus []model.AdminMenus
	if err := model.DB.Find(&menus).Error; err != nil {
		return res, err
	}
	for _, m := range menus {
		menuByCode[m.Code] = m
	}

	// 4. 对每条路由,查/插
	for _, route := range filtered {
		method := route.Method
		path := route.Path

		// 4.1 已存在?(按 method+path,包括软删行)
		//     ⚠️ 必须用 Unscoped():GORM 默认软删过滤会忽略 deleted_at IS NOT NULL 的行,
		//     但 UNIQUE 索引 idx_op_method_path 不看 deleted_at ——
		//     如果只靠默认 First 看活跃行,会判 "不存在" 然后 INSERT 触发 1062 Duplicate
		//     看到软删行就复活(deleted_at 置 NULL)而不是新建,跟"重新发现这条路由"语义一致
		var existing model.AdminMenuOperations
		err := model.DB.Unscoped().Where("method = ? AND path = ?", method, path).First(&existing).Error
		if err == nil {
			if existing.DeletedAt.Valid {
				// 软删行复活,同步给超管(因为后面第 5 步统一会处理新加的,
				// 这里只复活记录本身,role 关联见 GetByOperation 是否会顺带检查)
				if err := model.DB.Unscoped().Model(&existing).Update("deleted_at", nil).Error; err != nil {
					res.Skipped++
					res.SkipList = append(res.SkipList, method+" "+path+" (resurrect failed: "+err.Error()+")")
					continue
				}
			}
			res.Existing++
			continue
		}

		// 4.2 解析 module: /api/<code>/...  → code
		//     支持 /api/system/<code>/... 和 /api/codeDeploy/<code>/...
		pathAfterPrefix := strings.TrimPrefix(path, "/api/system/")
		pathAfterPrefix = strings.TrimPrefix(pathAfterPrefix, "/api/codeDeploy/")
		parts := strings.Split(pathAfterPrefix, "/")
		if len(parts) == 0 || parts[0] == "" {
			res.Skipped++
			res.SkipList = append(res.SkipList, method+" "+path+" (no module code in path)")
			continue
		}
		moduleCode := parts[0]
		menu, ok := menuByCode[moduleCode]
		if !ok {
			res.Skipped++
			res.SkipList = append(res.SkipList, method+" "+path+" (menu.code="+moduleCode+" not found)")
			continue
		}

		// 4.3 INSERT(默认 name=method+path,管理员可改)
		newOp := model.AdminMenuOperations{
			MenuID: menu.ID,
			Method: method,
			Path:   path,
			Name:   method + " " + path,
			Sort:   0,
		}
		if err := model.DB.Create(&newOp).Error; err != nil {
			// 唯一索引冲突(并发 sync 或重启) → 视为已存在,不计入 inserted
			if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "1062") {
				res.Existing++
				continue
			}
			logger.L.Error("SyncRoutes insert failed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Error(err),
			)
			res.Skipped++
			res.SkipList = append(res.SkipList, method+" "+path+" (insert err: "+err.Error()+")")
			continue
		}

		res.Inserted++
		res.InsertList = append(res.InsertList, method+" "+path)

		// 4.4 自动给超管角色关联上(不然没人能用这条新权限)
		if adminRoleID > 0 {
			// 先查是否已关联
			var cnt int64
			model.DB.Model(&model.AdminRoleOperations{}).
				Where("role_id = ? AND route_id = ?", adminRoleID, newOp.ID).
				Count(&cnt)
			if cnt == 0 {
				model.DB.Create(&model.AdminRoleOperations{
					RoleID:  adminRoleID,
					RouteID: newOp.ID,
				})
			}
		}
	}

	// 5. 有新增就 bump 超管版本,让 admin 重新登录或下次请求懒重载拿到新权限
	if res.Inserted > 0 && adminRoleID > 0 {
		go cache.BumpVersion(adminRoleID)
	}

	logger.L.Info("SyncRoutes done",
		zap.Int("total_filtered", res.Total),
		zap.Int("existing", res.Existing),
		zap.Int("inserted", res.Inserted),
		zap.Int("skipped", res.Skipped),
	)
	if len(res.InsertList) > 0 {
		logger.L.Info("SyncRoutes inserted", zap.Strings("list", res.InsertList))
	}
	if len(res.SkipList) > 0 {
		logger.L.Warn("SyncRoutes skipped", zap.Strings("list", res.SkipList))
	}

	return res, nil
}
