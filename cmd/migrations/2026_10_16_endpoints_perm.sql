-- 2026-10-16:为 GET /api/codeDeploy/endpoints 补 operation
-- 端字典接口,前端 ApkPackage.vue 在用;SyncRoutes 找不到 menu.code='endpoints' 会 skip

-- 1) operation(挂 menu_id=8 codeDeploy 顶级)
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (8, '端字典列表', 'GET', '/api/codeDeploy/endpoints', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

-- 2) 给超管 role_id=1 关联
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.method = 'GET' AND o.path = '/api/codeDeploy/endpoints';
