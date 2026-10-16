-- 2026-10-16:为「后端API部署」菜单(menu_id=9)加 git-deploy 接口权限
--   menu.code='online' 实际是 "后端API部署"(用户改过 name,code 仍是 online)
--   operation 挂 menu_id=9,超管 role_id=1 关联
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (9, '后端API部署(git reset)', 'POST', '/api/codeDeploy/assets/:id/git-deploy', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

-- 超管关联
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.method = 'POST' AND o.path = '/api/codeDeploy/assets/:id/git-deploy';
