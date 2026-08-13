-- ===========================================================
-- 代码部署(codeDeploy)模块 migration
--
-- 时间: 2026-08-13
-- 4 张表:端字典 / 业务项目 / 项目-端关联 / 代码包
--
-- 表前缀: config.AppConfig.Database.Prefix (默认空字符串)
--   实际表名: code_endpoints / business_projects /
--             business_project_endpoints / code_packages
--
-- 执行: mysql -u root -p <db> < 2026_08_13_code_deploy.sql
-- ===========================================================

SET NAMES utf8mb4;

-- ===========================================================
-- 1. 端字典表(4 个固定端,但走表便于扩展)
-- ===========================================================
CREATE TABLE IF NOT EXISTS `code_endpoints` (
  `id`         INT UNSIGNED  NOT NULL AUTO_INCREMENT COMMENT '主键',
  `code`       VARCHAR(32)   NOT NULL                COMMENT '机器名:ios/android/web/admin',
  `name`       VARCHAR(32)   NOT NULL                COMMENT '展示名:苹果/安卓/前台web/后台web',
  `ext`        VARCHAR(16)   NOT NULL                COMMENT '文件扩展名:apk/zip',
  `icon`       VARCHAR(64)   NOT NULL DEFAULT ''     COMMENT 'Element Plus icon 名',
  `sort`       INT           NOT NULL DEFAULT 0      COMMENT '排序',
  `status`     TINYINT       NOT NULL DEFAULT 1      COMMENT '1=启用 0=禁用',
  `created_at` DATETIME      NOT NULL,
  `updated_at` DATETIME      NOT NULL,
  `deleted_at` DATETIME      DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='代码包部署端字典';

INSERT INTO `code_endpoints` (`code`, `name`, `ext`, `icon`, `sort`, `status`, `created_at`, `updated_at`) VALUES
  ('ios',     '苹果',    'apk', 'Iphone',    10, 1, NOW(), NOW()),
  ('android', '安卓',    'apk', 'Cellphone', 20, 1, NOW(), NOW()),
  ('web',     '前台web', 'zip', 'Monitor',   30, 1, NOW(), NOW()),
  ('admin',   '后台web', 'zip', 'Setting',   40, 1, NOW(), NOW());


-- ===========================================================
-- 2. 业务项目表(单层,parent_id 留位备用)
-- ===========================================================
CREATE TABLE IF NOT EXISTS `business_projects` (
  `id`          INT UNSIGNED  NOT NULL AUTO_INCREMENT COMMENT '主键',
  `code`        VARCHAR(64)   NOT NULL                COMMENT '机器名:b2b/dianyao',
  `name`        VARCHAR(64)   NOT NULL                COMMENT '展示名:b2b电商项目',
  `description` VARCHAR(255)  NOT NULL DEFAULT '',
  `sort`        INT           NOT NULL DEFAULT 0,
  `status`      TINYINT       NOT NULL DEFAULT 1      COMMENT '1=启用 0=禁用',
  `created_at`  DATETIME      NOT NULL,
  `updated_at`  DATETIME      NOT NULL,
  `deleted_at`  DATETIME      DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_status_sort` (`status`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务项目';


-- ===========================================================
-- 3. 业务项目-端 关联表
-- ===========================================================
CREATE TABLE IF NOT EXISTS `business_project_endpoints` (
  `id`          INT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `project_id`  INT UNSIGNED  NOT NULL,
  `endpoint_id` INT UNSIGNED  NOT NULL,
  `sort`        INT           NOT NULL DEFAULT 0,
  `status`      TINYINT       NOT NULL DEFAULT 1      COMMENT '1=启用 0=禁用',
  `created_at`  DATETIME      NOT NULL,
  `updated_at`  DATETIME      NOT NULL,
  `deleted_at`  DATETIME      DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_endpoint` (`project_id`, `endpoint_id`),
  KEY `idx_project` (`project_id`),
  KEY `idx_endpoint` (`endpoint_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目下启用的端';


-- ===========================================================
-- 4. 代码包表
-- ===========================================================
CREATE TABLE IF NOT EXISTS `code_packages` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `project_id`   INT UNSIGNED    NOT NULL,
  `endpoint_id`  INT UNSIGNED    NOT NULL,
  `name`         VARCHAR(128)    NOT NULL             COMMENT '前端上传时填的原始名(不含版本号/扩展名)',
  `version`      VARCHAR(32)     NOT NULL             COMMENT '后端自动生成,v2.5.7',
  `full_name`    VARCHAR(255)    NOT NULL             COMMENT '拼接: name-version.ext',
  `ext`          VARCHAR(16)     NOT NULL             COMMENT '冗余存端 ext',
  `size`         BIGINT          NOT NULL DEFAULT 0   COMMENT '字节',
  `file_url`     VARCHAR(512)    NOT NULL             COMMENT 'OSS 访问地址',
  `file_path`    VARCHAR(255)    NOT NULL DEFAULT ''  COMMENT 'OSS 路径(便于删除)',
  `uploader_id`  INT UNSIGNED    NOT NULL             COMMENT '上传人 user_id',
  `build_time`   DATETIME        NOT NULL             COMMENT '用户填的构建时间',
  `note`         VARCHAR(500)    NOT NULL DEFAULT '',
  `status`       TINYINT         NOT NULL DEFAULT 1   COMMENT '1=有效 0=已下架',
  `created_at`   DATETIME        NOT NULL,
  `updated_at`   DATETIME        NOT NULL,
  `deleted_at`   DATETIME        DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_proj_ep_time` (`project_id`, `endpoint_id`, `build_time`),
  KEY `idx_uploader` (`uploader_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='代码包';
