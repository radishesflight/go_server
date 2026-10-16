SELECT '--- 超管 role_id=1 对 30/35/36 的关联 ---' AS '';
SELECT id, role_id, route_id FROM go_admin_role_operations WHERE route_id IN (30, 35, 36) AND role_id = 1;
SELECT '--- 超管当前 permissions(method+path) — 应有 30/35/36 的 path ---' AS '';
SELECT r.id AS role_op_id, r.role_id, o.id AS op_id, o.method, o.path, o.name
FROM go_admin_role_operations r
JOIN go_admin_menu_operations o ON o.id = r.route_id
WHERE r.role_id = 1 AND o.id IN (30, 35, 36) ORDER BY o.id;
