-- 2026-10-17:新建阿里云 ECS 镜像本地快照表 go_aliyun_images
--   用途:把镜像列表缓存到本地,避免每次列表查询都打 ECS API(12 秒)
--   同步策略:点"重新同步"按钮时,后端先删后插全量替换

USE go_server;

CREATE TABLE IF NOT EXISTS go_aliyun_images (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  image_id        VARCHAR(64)  NOT NULL,
  image_name      VARCHAR(255) NOT NULL,
  image_type      VARCHAR(16)  NOT NULL DEFAULT 'Self',
  image_owner_alias VARCHAR(32) NOT NULL DEFAULT '',
  region_id       VARCHAR(32)  NOT NULL,
  os_name         VARCHAR(255) NOT NULL DEFAULT '',
  architecture    VARCHAR(16)  NOT NULL DEFAULT '',
  size_gb         INT          NOT NULL DEFAULT 0,
  status          VARCHAR(32)  NOT NULL DEFAULT '',
  description     TEXT,
  snapshot_at     DATETIME     NOT NULL,
  created_at      DATETIME     NOT NULL,
  updated_at      DATETIME     NOT NULL,
  deleted_at      DATETIME     DEFAULT NULL,
  UNIQUE KEY uk_image_id (image_id),
  KEY idx_region_type (region_id, image_type),
  KEY idx_snapshot (snapshot_at),
  KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SELECT '--- after create ---' AS '';
DESCRIBE go_aliyun_images;
