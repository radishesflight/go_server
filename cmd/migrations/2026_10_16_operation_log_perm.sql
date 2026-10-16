-- 2026-10-16:为「部署日志」菜单建权限链路
-- menu code=operationLog 挂在 codeDeploy(menu_id=8)下
-- SyncRoutes 推断 moduleCode=operation-logs 没匹配,会 skip,手写补齐

-- 1) menu
INSERT INTO go_admin_menus
  (name, code, path, icon, parent_id, sort, status, created_at, updated_at)
VALUES
  ('部署日志', 'operationLog', '/codeDeploy/operationLog', 'Document', 8, 8, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 2) operation(GET /api/codeDeploy/operation-logs)
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (13, '操作日志列表', 'GET', '/api/codeDeploy/operation-logs', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

-- 3) operation(GET /api/codeDeploy/operation-logs/:id)
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (13, '操作日志详情', 'GET', '/api/codeDeploy/operation-logs/:id', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

-- 4) 给超管 role_id=1 关联
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.path IN ('/api/codeDeploy/operation-logs', '/api/codeDeploy/operation-logs/:id');
