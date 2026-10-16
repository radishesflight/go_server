-- 完整 UPDATE:把所有 codeDeploy/packages* 路径的 op 绑到 menu_id=14
UPDATE go_admin_menu_operations
SET menu_id = 14, updated_at = NOW()
WHERE path IN (
  '/api/codeDeploy/projects-with-paths',
  '/api/codeDeploy/packages',
  '/api/codeDeploy/packages/:id',
  '/api/codeDeploy/packages/:id/download-url'
) AND menu_id != 14;

-- role_op 关联(把刚绑的 op 关联超管)
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.menu_id = 14
  AND NOT EXISTS (
    SELECT 1 FROM go_admin_role_operations r
    WHERE r.role_id = 1 AND r.route_id = o.id
  );
