SELECT '--- menu ---' AS '';
SELECT id, name, code, parent_id, sort FROM go_admin_menus WHERE code = 'assets';
SELECT '--- operation ---' AS '';
SELECT id, menu_id, method, path FROM go_admin_menu_operations WHERE path = '/api/codeDeploy/assets/:id/probe-deps';
SELECT '--- role op ---' AS '';
SELECT r.id, r.role_id, o.method, o.path FROM go_admin_role_operations r JOIN go_admin_menu_operations o ON o.id = r.route_id WHERE o.path = '/api/codeDeploy/assets/:id/probe-deps';
