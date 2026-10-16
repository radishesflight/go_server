-- 补救:把已有 operations 绑到 menu_id=14
-- 之前 INSERT ON DUPLICATE KEY UPDATE 没把 menu_id 加进 UPDATE,导致 path+method 唯一冲突时只更新 name 不绑 menu
UPDATE go_admin_menu_operations
SET menu_id = 14, updated_at = NOW()
WHERE path IN (
  '/api/codeDeploy/projects-with-paths',
  '/api/codeDeploy/packages',
  '/api/codeDeploy/packages/:id/download-url'
) AND menu_id != 14;

-- 超管 role_op 关联(如果之前没绑上,这次补)
INSERT IGNORE INTO go_admin_role_operations (role_id, route_id)
SELECT 1, o.id FROM go_admin_menu_operations o
WHERE o.menu_id = 14
  AND NOT EXISTS (
    SELECT 1 FROM go_admin_role_operations r
    WHERE r.role_id = 1 AND r.route_id = o.id
  );
