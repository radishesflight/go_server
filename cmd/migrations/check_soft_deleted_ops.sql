SELECT '--- 软删的 menu_operations (deleted_at 非空) ---' AS '';
SELECT id, menu_id, method, path, name, deleted_at FROM go_admin_menu_operations WHERE deleted_at IS NOT NULL ORDER BY id DESC LIMIT 10;
SELECT '--- 活跃的同 method+path (会跟软删行冲突 INSERT) ---' AS '';
SELECT id, menu_id, method, path FROM go_admin_menu_operations WHERE (method, path) IN (('PUT', '/api/codeDeploy/projects/:id'), ('DELETE', '/api/codeDeploy/projects/:id'), ('GET', '/api/codeDeploy/endpoints')) AND deleted_at IS NULL ORDER BY method, path;
