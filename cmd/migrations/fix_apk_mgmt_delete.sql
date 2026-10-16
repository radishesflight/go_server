-- 补 DELETE op(原 SyncRoutes 跳过了,需要 INSERT)
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (14, 'APK包删除', 'DELETE', '/api/codeDeploy/packages/:id', NOW(), NOW());

-- 关联超管
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, id FROM go_admin_menu_operations
WHERE menu_id = 14 AND method = 'DELETE' AND path = '/api/codeDeploy/packages/:id';
