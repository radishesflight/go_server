SELECT '--- operation ---' AS '';
SELECT id, method, path, name FROM go_admin_menu_operations WHERE path LIKE '%download-url%';
SELECT '--- 超管关联 ---' AS '';
SELECT r.id AS rel_id, r.role_id, o.method, o.path FROM go_admin_role_operations r
JOIN go_admin_menu_operations o ON o.id = r.route_id
WHERE o.path LIKE '%download-url%' AND r.role_id = 1;
