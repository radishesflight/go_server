-- 2026-10-17 新增云平台凭证管理子菜单
--   表:go_cloud_credentials(阿里云/腾讯云 AccessKey/SecretKey)
--   菜单:云平台凭证(code=credentials,parent=15)
--   SyncRoutes 自动按 path 段 "credentials" 反查 menu.code=credentials 注册 op
--   再手工绑 role_id=1(超管)关联菜单,常规做法
USE go_server;

-- 1. 建表
CREATE TABLE IF NOT EXISTS go_cloud_credentials (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(100) NOT NULL                  COMMENT '备注名(同一厂商内唯一,如"xxx公司")',
  type          VARCHAR(16)  NOT NULL                  COMMENT 'aliyun / tencent',
  access_key    VARCHAR(128) NOT NULL                  COMMENT 'AK(公钥)',
  secret_key    VARCHAR(255) NOT NULL                  COMMENT 'SK(私钥)',
  remark        VARCHAR(500) DEFAULT NULL              COMMENT '备注',
  created_at    DATETIME     NOT NULL,
  updated_at    DATETIME     NOT NULL,
  deleted_at    DATETIME     DEFAULT NULL,
  UNIQUE KEY uk_name_type (name, type),
  KEY idx_type (type),
  KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SELECT '--- after create ---' AS '';
DESCRIBE go_cloud_credentials;

-- 2. 插菜单
INSERT INTO go_admin_menus
  (name, code, path, icon, parent_id, sort, status, data_scope, created_at, updated_at)
VALUES
  ('云平台凭证', 'credentials', '/serverDeploy/cloudCredential', 'Key', 15, 2, 1, 0, NOW(), NOW());

-- 3. 绑超管 role_id=1(admin_role_menus 只有 id/role_id/menu_id 三列)
INSERT INTO go_admin_role_menus (role_id, menu_id)
SELECT 1, id FROM go_admin_menus WHERE code = 'credentials';

-- 4. 复核
SELECT m.id, m.parent_id, m.name, m.code, m.path FROM go_admin_menus WHERE code = 'credentials';
SELECT r.role_id, m.code FROM go_admin_role_menus r
  JOIN go_admin_menus m ON m.id = r.menu_id
  WHERE m.code = 'credentials';
