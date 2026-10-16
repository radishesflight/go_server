SELECT '--- menu ---' AS '';
SELECT id, name, code, parent_id, path, icon FROM go_admin_menus WHERE code = 'apkManagement';
SELECT '--- operations (menu_id=14) ---' AS '';
SELECT id, menu_id, method, path, name FROM go_admin_menu_operations WHERE menu_id = 14 ORDER BY id;
SELECT '--- role_menus (apkMgmt) ---' AS '';
SELECT role_id, menu_id FROM go_admin_role_menus WHERE menu_id = 14;
