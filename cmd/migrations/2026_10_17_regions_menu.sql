-- 2026-10-17:给 serverDeploy 加 regions 子菜单(SyncRoutes 反查用,前端不直接展示)
--   路径:/api/serverDeploy/regions (GET 列可用 region)
--   SyncRoutes 按 path 第二段 imageList/regions 反查 menu.code

USE go_server;

INSERT INTO go_admin_menus (name, code, path, icon, parent_id, sort, status, data_scope, created_at, updated_at)
VALUES ('可用区域', 'regions', '/serverDeploy/regions', '', 15, 1, 1, 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

INSERT IGNORE INTO go_admin_role_menus (role_id, menu_id)
SELECT 1, m.id FROM go_admin_menus m WHERE m.code = 'regions';

SELECT '--- after insert ---' AS '';
SELECT id, name, code, parent_id, sort FROM go_admin_menus WHERE code IN ('imageList', 'regions');
SELECT '--- role_menus ---' AS '';
SELECT r.role_id, m.code, m.name FROM go_admin_role_menus r JOIN go_admin_menus m ON m.id = r.menu_id WHERE m.code IN ('imageList', 'regions');
