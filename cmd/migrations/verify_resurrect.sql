SELECT '--- 之前软删的 3 行(应该已被复活,deleted_at 为 NULL) ---' AS '';
SELECT id, menu_id, method, path, name, deleted_at FROM go_admin_menu_operations WHERE id IN (30, 35, 36) ORDER BY id;
SELECT '--- 仍软删的 operations(应该没有了) ---' AS '';
SELECT COUNT(*) AS soft_deleted_count FROM go_admin_menu_operations WHERE deleted_at IS NOT NULL;
