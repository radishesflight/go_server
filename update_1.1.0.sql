-- =====================================================
-- go_server 1.1.0 增量升级 SQL
-- 用途:登录失败 3 次自动锁定 + admin 手动解锁功能
-- 适用:已上线 1.0.x 环境升级到 1.1.0
-- 数据库:go_server  (库名跟 .env 里的 MYSQL_DBNAME 一致)
-- =====================================================
--
-- 跑法(三选一):
--   1. mysql -h<host> -P13306 -uroot -p123456 go_server < update_1.1.0.sql
--   2. docker exec -i go_server_mysql mysql -uroot -p123456 go_server < update_1.1.0.sql
--   3. docker exec go_server_mysql mysql -uroot -p123456 go_server -e "$(cat update_1.1.0.sql)"
--
-- 注意:
--   - 本 SQL 是 fallback,goc_server 1.1.0 启动时 AutoMigrate 会自动加列,
--     SyncRoutes 会自动同步 unlock op;但如果你想跳过重启验证,先跑这个也行
--   - 如果你生产 MySQL < 8.0,把 ALTER TABLE 那行去掉 ADD COLUMN IF NOT EXISTS
--     (MySQL 8.0.16+ 才支持),改用 INFORMATION_SCHEMA 判断或先 SHOW COLUMNS 看看
-- =====================================================

-- -----------------------------------------------------
-- 1. admin_users 表加两个新列(登录限制用)
-- -----------------------------------------------------
ALTER TABLE `go_admin_users`
  ADD COLUMN `failed_count` bigint NOT NULL DEFAULT 0 COMMENT '连续登录失败次数,达 3 次自动锁定',
  ADD COLUMN `locked_at`    datetime(3) NULL DEFAULT NULL COMMENT '锁定时间,NULL=未锁定';

-- -----------------------------------------------------
-- 2. 加 unlock 操作 op(POST /api/system/adminUsers/unlock/:id)
--    menu_id=3 是 adminUsers 菜单(来自 seed)
--    SyncRoutes 启动时也会自动同步,这条是双保险
-- -----------------------------------------------------
INSERT IGNORE INTO `go_admin_menu_operations`
  (`id`, `menu_id`, `method`, `path`, `name`, `sort`, `created_at`, `updated_at`)
VALUES
  (NULL, 3, 'POST', '/api/system/adminUsers/unlock/:id', '解除登录限制', 6, NOW(), NOW());

-- -----------------------------------------------------
-- 3. 给超管角色(role_id=1)授权 unlock op
--    INSERT IGNORE 防主键冲突
-- -----------------------------------------------------
SET @unlock_route_id = (
  SELECT id FROM `go_admin_menu_operations`
  WHERE `path` = '/api/system/adminUsers/unlock/:id' LIMIT 1
);
INSERT IGNORE INTO `go_admin_role_operations` (`role_id`, `route_id`)
VALUES (1, @unlock_route_id);

-- -----------------------------------------------------
-- 4. (可选)如果你有其他角色也需要解锁权限,把 role_id 替换:
--    INSERT IGNORE INTO `go_admin_role_operations` (`role_id`, `route_id`)
--    VALUES (<your_role_id>, @unlock_route_id);
-- -----------------------------------------------------

-- -----------------------------------------------------
-- 5. 验证:跑完应该看到
--    - DESCRIBE go_admin_users; 多出 failed_count + locked_at 两列
--    - SELECT * FROM go_admin_menu_operations WHERE path LIKE '%unlock%'; 有一条
--    - SELECT * FROM go_admin_role_operations WHERE route_id=@unlock_route_id; 有超管那条
-- -----------------------------------------------------

-- =====================================================
-- 配套部署步骤(给你生产的 .env 和 docker-compose.yml)
-- =====================================================
--
-- 1. 把本地 F:\work\my\go_server\docker-compose.yml 传到生产(已改 image tag → 1.1.0):
--    scp docker-compose.yml root@<生产主机>:/root/ces/docker-compose.yml
--
-- 2. 生产上:
--    cd /root/ces
--    docker compose pull              # 拉新镜像 go_server:1.1.0 + web:1.1.0
--    docker compose up -d             # 用新镜像重启(数据保留,mysql_data volume 不动)
--    docker ps                        # 看 4 容器都 healthy
--
-- 3. (可选)如果你先跑了这个 SQL,go_server 启动时 AutoMigrate 会跳过(列已存在)
--    如果没跑,AutoMigrate 自动加列
-- =====================================================