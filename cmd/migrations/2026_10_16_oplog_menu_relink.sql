-- 2026-10-16 补救:把"部署日志"菜单(menu_id=13)绑给超管 role_id=1
-- 之前 2026_10_16_operation_log_perm.sql 只补了 operation + role_op 关联,
-- 漏了 role_menus 关联,导致前端 SideBar 看不到菜单入口
-- 用 INSERT IGNORE 防止重复插入
INSERT IGNORE INTO go_admin_role_menus (role_id, menu_id)
VALUES (1, 13);
