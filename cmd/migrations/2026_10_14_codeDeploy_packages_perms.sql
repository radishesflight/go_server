-- 补 codeDeploy/packages 全部接口的 operation + 超管授权
-- 背景:SyncRoutes 因 admin_menus.code='packages' 不存在,自动 INSERT 失败,这里手写
USE go_server;

-- 挂在"apk/前端包管理"菜单(menu_id=10,code='apkPackage')下,前端"上传代码包"页面同模块
INSERT IGNORE INTO go_admin_menu_operations
  (created_at, updated_at, deleted_at, menu_id, method, path, name, sort)
VALUES
  (NOW(), NOW(), NULL, 10, 'GET',    '/api/codeDeploy/packages',                  '代码包列表', 0),
  (NOW(), NOW(), NULL, 10, 'GET',    '/api/codeDeploy/packages/:id',              '代码包详情', 0),
  (NOW(), NOW(), NULL, 10, 'POST',   '/api/codeDeploy/packages',                  '上传代码包', 0),
  (NOW(), NOW(), NULL, 10, 'POST',   '/api/codeDeploy/packages/:id/pull',         '触发部署', 0),
  (NOW(), NOW(), NULL, 10, 'DELETE', '/api/codeDeploy/packages/:id',              '删除代码包', 0),
  (NOW(), NOW(), NULL, 10, 'GET',    '/api/codeDeploy/packages/:id/download-url', '代码包下载链接', 0);

-- 给超管(role_id=1) 关联
INSERT INTO go_admin_role_operations (role_id, route_id)
SELECT 1, id FROM go_admin_menu_operations
WHERE path IN (
  '/api/codeDeploy/packages',
  '/api/codeDeploy/packages/:id',
  '/api/codeDeploy/packages/:id/pull',
  '/api/codeDeploy/packages/:id/download-url'
);

-- 自校验
SELECT 'op' AS k, id, method, path, name FROM go_admin_menu_operations
WHERE path IN (
  '/api/codeDeploy/packages',
  '/api/codeDeploy/packages/:id',
  '/api/codeDeploy/packages/:id/pull',
  '/api/codeDeploy/packages/:id/download-url'
) ORDER BY id;
SELECT 'role' AS k, role_id, route_id FROM go_admin_role_operations
WHERE route_id IN (
  SELECT id FROM go_admin_menu_operations
  WHERE path IN (
    '/api/codeDeploy/packages',
    '/api/codeDeploy/packages/:id',
    '/api/codeDeploy/packages/:id/pull',
    '/api/codeDeploy/packages/:id/download-url'
  )
);
