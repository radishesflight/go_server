-- 2026-10-16:补回超管对复活 operations 的关联
-- 背景:之前 3 条 operation(id=30/35/36)被软删,SyncRoutes 启动时复活了但 role_operations 表关联丢了
--      这意味着 admin/123456 登录后 token 里没这 3 条权限,PATCH/DELETE projects 会 2001 NoPermission
-- 补救:重新插入关联(idempotent:已存在则跳过)

INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id
FROM go_admin_menu_operations o
WHERE o.id IN (30, 35, 36);
