-- 1) 菜单实际状态
SELECT '--- menu ---' AS '';
SELECT id, name, code, path, icon, parent_id, sort, status FROM go_admin_menus WHERE code = 'operationLog' OR name = '部署日志';
-- 2) 父菜单(codeDeploy)下的所有子菜单
SELECT '--- codeDeploy children ---' AS '';
SELECT id, name, code, path, sort, status FROM go_admin_menus WHERE parent_id = 8 ORDER BY sort;
-- 3) 超管 role_id=1 的所有可见菜单
SELECT '--- role_id=1 menus ---' AS '';
SELECT r.menu_id, m.name, m.code, m.path, m.status FROM go_admin_role_menus r JOIN go_admin_menus m ON m.id = r.menu_id WHERE r.role_id = 1 ORDER BY r.menu_id;
-- 4) admin_users 缓存的 menus/permissions(看 token 里的)
SELECT '--- admin user token payload (c.Set menus) ---' AS '';
SELECT id, username FROM go_admin_users WHERE username = 'admin';
