SELECT '--- menu ---' AS '';
SELECT id, name, code, path, parent_id, sort FROM go_admin_menus WHERE code = 'operationLog';
SELECT '--- operations ---' AS '';
SELECT id, menu_id, method, path, name FROM go_admin_menu_operations WHERE path LIKE '%operation-logs%' ORDER BY id;
SELECT '--- role ops ---' AS '';
SELECT r.id, r.role_id, o.method, o.path FROM go_admin_role_operations r
JOIN go_admin_menu_operations o ON o.id = r.route_id
WHERE o.path LIKE '%operation-logs%' AND r.role_id = 1;
