-- 2026-10-16:为新增的 GET /api/codeDeploy/projects-with-paths 接口补 operation
-- (SyncRoutes 找不到 menu.code='projects-with-paths' 会 skip,手写补)
-- 挂到 menu_id=10 (apk/前端包管理) 下,然后给超管 role_id=1 关联

-- 1) operation 记录
INSERT INTO go_admin_menu_operations
  (menu_id, name, method, path, created_at, updated_at)
VALUES
  (10, '项目+资产+路径树', 'GET', '/api/codeDeploy/projects-with-paths', NOW(), NOW())
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  updated_at = NOW();

-- 2) 给超管 role_id=1 关联(让 admin 用户能访问)
--    go_admin_role_operations 表无 created_at/updated_at 字段
INSERT INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id
FROM go_admin_menu_operations o
WHERE o.method = 'GET' AND o.path = '/api/codeDeploy/projects-with-paths'
  AND NOT EXISTS (
    SELECT 1 FROM go_admin_role_operations r
    WHERE r.role_id = 1 AND r.route_id = o.id
  );
