SELECT '--- operations for operation-logs ---' AS '';
SELECT id, menu_id, method, path, name FROM go_admin_menu_operations WHERE path LIKE '%operation-logs%' ORDER BY id;
SELECT '--- role_op associations (admin) ---' AS '';
SELECT r.id AS rel_id, r.role_id, o.id AS op_id, o.method, o.path FROM go_admin_role_operations r JOIN go_admin_menu_operations o ON o.id = r.route_id WHERE r.role_id = 1 AND o.path LIKE '%operation-logs%';
