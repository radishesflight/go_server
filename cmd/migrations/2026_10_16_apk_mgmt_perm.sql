-- 2026-10-16:为「APK包管理」菜单(menu_id=14)建权限链路
-- 路径:/codeDeploy/apkManagement,基于 ApkPackage.vue 复制,删了部署按钮(APK 不走服务器部署)

-- 1) menu
INSERT INTO go_admin_menus
  (name, code, path, icon, parent_id, sort, status, created_at, updated_at)
VALUES
  ('APK包管理', 'apkManagement', '/codeDeploy/apkManagement', 'Cellphone', 8, 14, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 2) operation(列表权限)— APK 只看/下载/删除包,不需要部署
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (14, 'APK包列表', 'GET', '/api/codeDeploy/projects-with-paths', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (14, 'APK包列表', 'GET', '/api/codeDeploy/packages', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (14, 'APK包上传', 'POST', '/api/codeDeploy/packages', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (14, 'APK包删除', 'DELETE', '/api/codeDeploy/packages/:id', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (14, 'APK包下载', 'GET', '/api/codeDeploy/packages/:id/download-url', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = NOW();

-- 3) 给超管 role_id=1 关联
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.path IN (
  '/api/codeDeploy/projects-with-paths',
  '/api/codeDeploy/packages',
  '/api/codeDeploy/packages/:id/download-url'
);

-- 4) role_menus 关联
INSERT IGNORE INTO go_admin_role_menus (role_id, menu_id)
VALUES (1, 14);
