-- 2026-10-17 新增云服务器列表持久化
--   - 表 go_cloud_servers(各厂商实例快照)
--   - 菜单 id=20 code=servers parent_id=15
--   - SyncRoutes 按 path 段 "servers" 反查 menu.code=servers 自动注册 op
--   - 手工绑超管 role_id=1
USE go_server;

-- 1. 建表
CREATE TABLE IF NOT EXISTS go_cloud_servers (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  credential_id  BIGINT UNSIGNED NOT NULL                COMMENT '关联的凭证 id',
  vendor         VARCHAR(16)  NOT NULL                  COMMENT 'aliyun / tencent',
  instance_id    VARCHAR(64)  NOT NULL                  COMMENT '云厂商实例 ID',
  name           VARCHAR(200) NOT NULL                  COMMENT '实例名',
  region         VARCHAR(64)  NOT NULL                  COMMENT '所在 region',
  zone           VARCHAR(64)  DEFAULT ''               COMMENT '所在 zone(可选)',
  instance_type  VARCHAR(64)  DEFAULT ''               COMMENT '规格(ecs.g6.large)',
  status         VARCHAR(32)  DEFAULT ''               COMMENT 'Running / Stopped 等',
  private_ip     VARCHAR(64)  DEFAULT ''               COMMENT '内网 IP',
  public_ip      VARCHAR(64)  DEFAULT ''               COMMENT '公网 IP(可空)',
  os_name        VARCHAR(255) DEFAULT ''               COMMENT '系统显示名',
  cpu            INT          NOT NULL DEFAULT 0,
  memory_gb      INT          NOT NULL DEFAULT 0,
  instance_created_at DATETIME DEFAULT NULL            COMMENT '云厂商上的创建时间',
  expired_at     DATETIME DEFAULT NULL                  COMMENT '到期时间(可空)',
  snapshot_at    DATETIME     NOT NULL                  COMMENT '本次拉取时间',
  created_at     DATETIME     NOT NULL,
  updated_at     DATETIME     NOT NULL,
  deleted_at     DATETIME     DEFAULT NULL,
  UNIQUE KEY uk_vendor_instance (vendor, instance_id),
  KEY idx_credential (credential_id),
  KEY idx_region (region),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SELECT '--- after create ---' AS '';
DESCRIBE go_cloud_servers;

-- 2. 插菜单(代码段跟 path /servers 对齐,SyncRoutes 自动绑 op)
INSERT INTO go_admin_menus
  (name, code, path, icon, parent_id, sort, status, data_scope, created_at, updated_at)
VALUES
  ('云服务器列表', 'servers', '/serverDeploy/serverList', 'Monitor', 15, 3, 1, 0, NOW(), NOW());

-- 3. 绑超管 role_id=1
INSERT IGNORE INTO go_admin_role_menus (role_id, menu_id)
SELECT 1, id FROM go_admin_menus WHERE code = 'servers';

-- 4. 复核
SELECT m.id, m.parent_id, m.name, m.code, m.path FROM go_admin_menus WHERE code = 'servers';
SELECT role_id, menu_id FROM go_admin_role_menus WHERE menu_id IN (SELECT id FROM go_admin_menus WHERE code = 'servers');
