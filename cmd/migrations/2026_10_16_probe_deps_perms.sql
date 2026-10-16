-- 2026-10-16:为新接口 POST /api/codeDeploy/assets/:id/probe-deps 建权限链路
-- 路径第一段 assets 没对应 menu,SyncRoutes 会 skip,手写补齐

-- 1) menu:code=assets 挂在 codeDeploy(menu_id=8)下
INSERT INTO go_admin_menus
  (name, code, path, icon, parent_id, sort, status, created_at, updated_at)
VALUES
  ('资产', 'assets', '/codeDeploy/assets', 'Coin', 8, 5, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 2) operation:依赖探测
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (12, '资产依赖探测', 'POST', '/api/codeDeploy/assets/:id/probe-deps', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

-- 3) 给超管 role_id=1 关联
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.method = 'POST' AND o.path = '/api/codeDeploy/assets/:id/probe-deps';
