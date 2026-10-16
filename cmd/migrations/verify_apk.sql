SELECT '--- 新增 operation ---' AS '';
SELECT id, menu_id, method, path, name FROM go_admin_menu_operations WHERE path = '/api/codeDeploy/projects-with-paths';
SELECT '--- 超管关联 ---' AS '';
SELECT r.id, r.role_id, r.route_id, o.method, o.path
FROM go_admin_role_operations r
JOIN go_admin_menu_operations o ON o.id = r.route_id
WHERE o.path = '/api/codeDeploy/projects-with-paths';
