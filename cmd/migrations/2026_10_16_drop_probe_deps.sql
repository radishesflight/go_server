-- 2026-10-16:反向迁移 — 撤销资产依赖探测接口的权限链路
--   原因:probe-deps 前后端均无调用,前端未接入口,清理死代码
--   步骤:
--     1. 删 admin_role_operations 中绑到 probe-deps op 的关联(防御性,本次实际无记录)
--     2. 删 admin_menu_operations 中 probe-deps op(id=103)
--     3. menu "资产"(id=12)保留 — 它还被其他 op 用着

USE go_server;

-- 1. 删 role_op 关联(如果有)
DELETE r FROM go_admin_role_operations r
JOIN go_admin_menu_operations o ON o.id = r.route_id
WHERE o.path = '/api/codeDeploy/assets/:id/probe-deps';

-- 2. 删 op 本身
DELETE FROM go_admin_menu_operations
WHERE path = '/api/codeDeploy/assets/:id/probe-deps';

-- 验证
SELECT '--- after cleanup ---' AS '';
SELECT COUNT(*) AS remaining_probe_ops
FROM go_admin_menu_operations
WHERE path = '/api/codeDeploy/assets/:id/probe-deps';
