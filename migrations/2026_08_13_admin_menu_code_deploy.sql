-- ===========================================================
-- 给 codeDeploy 模块补 admin_menus 菜单记录
--
-- 背景:
--   SyncRoutes 通过 /api/<module>/... 的 module 段反查 admin_menus.code
--   admin_menus 没有 code='codeDeploy' 这条记录 → SyncRoutes 跳过所有
--   /api/codeDeploy/* 路由 → 没有任何用户(即使超管)有权限调
--
-- 后续流程(临时没启权限,不需要跑这条):
--   1. 跑本 SQL,插入一条 code='codeDeploy' 菜单
--   2. 改 route_sync.go::SyncRoutes,前缀白名单加 "/api/codeDeploy/"
--   3. 重启后端,SyncRoutes 会自动 INSERT 11 条 operation + 给 role_id=1 授权
--   4. 改回 router 的 PermissionMiddleware 中间件
-- ===========================================================

-- codeDeploy 父菜单(parent_id 0 = 顶级,跟"系统管理"平级)
INSERT INTO `admin_menus` (`name`, `code`, `path`, `icon`, `parent_id`, `sort`, `status`, `data_scope`, `created_at`, `updated_at`)
VALUES ('代码部署', 'codeDeploy', '/codeDeploy', 'Upload', 0, 85, 1, 0, NOW(), NOW());
