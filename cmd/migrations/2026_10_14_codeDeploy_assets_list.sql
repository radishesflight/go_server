-- 给超管补"全部资产"接口权限
USE go_server;

-- 1) 新 operation:GET /api/codeDeploy/assets
-- 挂在"资产列表"菜单下(menu_id=11,code='projects'),
-- 这样前端"资产列表"页的"代码拉取控制台"按钮组能一起显示
INSERT INTO go_admin_menu_operations
  (created_at, updated_at, deleted_at, menu_id, method, path, name, sort)
VALUES
  (NOW(), NOW(), NULL, 11, 'GET', '/api/codeDeploy/assets', '查询全部资产', 0);

-- 2) 给超管(role_id=1) 关联这条 operation
INSERT INTO go_admin_role_operations (role_id, route_id)
SELECT 1, id FROM go_admin_menu_operations
WHERE method='GET' AND path='/api/codeDeploy/assets';

-- 3) 自校验
SELECT 'op' AS k, id, method, path, name, menu_id
FROM go_admin_menu_operations
WHERE path='/api/codeDeploy/assets';
SELECT 'role' AS k, role_id, route_id
FROM go_admin_role_operations
WHERE route_id IN (SELECT id FROM go_admin_menu_operations WHERE path='/api/codeDeploy/assets');
