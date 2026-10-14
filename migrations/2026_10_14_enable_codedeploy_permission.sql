-- ===========================================================
-- 启用 codeDeploy 模块权限(配合改动 internal/router/codeDeploy/codeDeploy.go
-- 和 internal/service/route_sync.go)
--
-- 背景:
--   codeDeploy 路由之前 PermissionMiddleware 被注释掉,任何登录用户可访问。
--   现在:
--     1. codeDeploy.go 已在所有路由上挂 middleware.PermissionMiddleware()
--     2. route_sync.go 的白名单已加 /api/codeDeploy/
--     3. 重启后端,SyncRoutes 会自动 INSERT 11 条 operation(去重后)
--        + 给超管(role_id=1)关联上
--     4. 跑本 SQL 补上 admin_menus 的 code='codeDeploy' 记录
--        (SyncRoutes 找不到 menu 就 skip,必须先有菜单)
--
-- 跑法:
--   mysql -h192.168.186.200 -uroot -p123456 go_admin < this.sql
-- ===========================================================

INSERT INTO `admin_menus` (`name`, `code`, `path`, `icon`, `parent_id`, `sort`, `status`, `data_scope`, `created_at`, `updated_at`)
VALUES ('代码部署', 'codeDeploy', '/codeDeploy', 'Upload', 0, 85, 1, 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE `updated_at` = NOW();
