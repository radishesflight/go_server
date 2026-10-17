-- 2026-10-17:新建顶级菜单"服务器部署" + 子菜单"镜像列表"
--   用途:阿里云 ECS 自定义镜像管理(共享/复制/列表)
--   顶级菜单 sort=10(代码部署 0 和 系统管理 90 之间)
--   子菜单 sort=0
--
-- 注意(2026-10-17 踩坑):
--   多行 VALUES INSERT 时 LAST_INSERT_ID() 返回的是**第一条**的 ID,
--   但某些 MySQL 版本/驱动在多行 batch INSERT 后会返回 0。
--   稳妥做法:父菜单独立 INSERT,拿到 ID 后子菜单用变量/已知 ID。

USE go_server;

-- 1. 顶级菜单"服务器部署"(必须先单独插,后面才能稳定拿 ID)
INSERT INTO go_admin_menus (name, code, path, icon, parent_id, sort, status, data_scope, created_at, updated_at)
VALUES ('服务器部署', 'serverDeploy', '/serverDeploy', 'Monitor', 0, 10, 1, 0, NOW(), NOW());

-- 2. 子菜单"镜像列表"(用上一次 INSERT 的 ID 作为 parent_id)
SET @parent_id = LAST_INSERT_ID();
INSERT INTO go_admin_menus (name, code, path, icon, parent_id, sort, status, data_scope, created_at, updated_at)
VALUES ('镜像列表', 'imageList', '/serverDeploy/imageList', '', @parent_id, 0, 1, 0, NOW(), NOW());

-- 3. 给超管(role_id=1)授权这两个菜单
INSERT INTO go_admin_role_menus (role_id, menu_id)
SELECT 1, m.id FROM go_admin_menus m
WHERE m.code IN ('serverDeploy', 'imageList')
ON DUPLICATE KEY UPDATE menu_id = menu_id;

-- 验证
SELECT '--- after insert ---' AS '';
SELECT id, name, code, icon, parent_id, sort FROM go_admin_menus WHERE code IN ('serverDeploy', 'imageList');
SELECT '--- role_menus ---' AS '';
SELECT r.role_id, m.name, m.code FROM go_admin_role_menus r JOIN go_admin_menus m ON m.id = r.menu_id WHERE m.code IN ('serverDeploy', 'imageList');
