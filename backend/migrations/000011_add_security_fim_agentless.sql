CREATE TABLE IF NOT EXISTS `security_fim_policies` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(120) NOT NULL COMMENT '策略名称',
  `description` VARCHAR(500) DEFAULT NULL COMMENT '策略描述',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `severity` VARCHAR(20) NOT NULL DEFAULT 'high' COMMENT '默认告警级别',
  `scan_interval_sec` INT NOT NULL DEFAULT 300 COMMENT '扫描周期秒',
  `hash_mode` VARCHAR(20) NOT NULL DEFAULT 'changed_only' COMMENT 'off/changed_only/full',
  `compare_mode` VARCHAR(20) NOT NULL DEFAULT 'baseline' COMMENT 'baseline/last_snapshot',
  `created_by` VARCHAR(100) DEFAULT NULL,
  `updated_by` VARCHAR(100) DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_security_fim_policies_name` (`name`),
  KEY `idx_security_fim_policies_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件完整性巡检策略';

CREATE TABLE IF NOT EXISTS `security_fim_policy_targets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `policy_id` BIGINT UNSIGNED NOT NULL,
  `server_id` BIGINT UNSIGNED NOT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `last_scan_at` DATETIME DEFAULT NULL,
  `last_scan_status` VARCHAR(20) DEFAULT NULL COMMENT 'success/failed/running',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_security_fim_policy_targets` (`policy_id`, `server_id`),
  KEY `idx_security_fim_policy_targets_server_id` (`server_id`),
  CONSTRAINT `fk_security_fim_policy_targets_policy`
    FOREIGN KEY (`policy_id`) REFERENCES `security_fim_policies` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巡检策略绑定主机';

CREATE TABLE IF NOT EXISTS `security_fim_watch_paths` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `policy_id` BIGINT UNSIGNED NOT NULL,
  `path` VARCHAR(500) NOT NULL COMMENT '监控目录',
  `recursive` TINYINT(1) NOT NULL DEFAULT 1,
  `max_depth` INT NOT NULL DEFAULT 0 COMMENT '0不限',
  `file_glob` VARCHAR(255) DEFAULT NULL,
  `exclude_glob` VARCHAR(255) DEFAULT NULL,
  `hash_on_match_only` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_security_fim_watch_paths_policy_id` (`policy_id`),
  CONSTRAINT `fk_security_fim_watch_paths_policy`
    FOREIGN KEY (`policy_id`) REFERENCES `security_fim_policies` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巡检目录配置';

CREATE TABLE IF NOT EXISTS `security_fim_snapshots` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `policy_id` BIGINT UNSIGNED NOT NULL,
  `server_id` BIGINT UNSIGNED NOT NULL,
  `snapshot_type` VARCHAR(20) NOT NULL DEFAULT 'scheduled' COMMENT 'baseline/scheduled/manual',
  `status` VARCHAR(20) NOT NULL DEFAULT 'running' COMMENT 'running/success/failed',
  `started_at` DATETIME NOT NULL,
  `finished_at` DATETIME DEFAULT NULL,
  `entry_count` BIGINT NOT NULL DEFAULT 0,
  `error_message` TEXT DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_security_fim_snapshots_policy_server` (`policy_id`, `server_id`),
  KEY `idx_security_fim_snapshots_status` (`status`),
  KEY `idx_security_fim_snapshots_started_at` (`started_at`),
  CONSTRAINT `fk_security_fim_snapshots_policy`
    FOREIGN KEY (`policy_id`) REFERENCES `security_fim_policies` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巡检快照';

CREATE TABLE IF NOT EXISTS `security_fim_snapshot_entries` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `snapshot_id` BIGINT UNSIGNED NOT NULL,
  `path` VARCHAR(1000) NOT NULL,
  `entry_type` VARCHAR(20) NOT NULL DEFAULT 'file' COMMENT 'file/dir/symlink',
  `size` BIGINT NOT NULL DEFAULT 0,
  `mode` VARCHAR(16) DEFAULT NULL,
  `owner` VARCHAR(100) DEFAULT NULL,
  `group_name` VARCHAR(100) DEFAULT NULL,
  `mtime` DATETIME DEFAULT NULL,
  `sha256` CHAR(64) DEFAULT NULL,
  `target_path` VARCHAR(1000) DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_security_fim_snapshot_entries` (`snapshot_id`, `path`(255)),
  KEY `idx_security_fim_snapshot_entries_path` (`path`(255)),
  CONSTRAINT `fk_security_fim_snapshot_entries_snapshot`
    FOREIGN KEY (`snapshot_id`) REFERENCES `security_fim_snapshots` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巡检快照明细';

CREATE TABLE IF NOT EXISTS `security_fim_diff_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `policy_id` BIGINT UNSIGNED NOT NULL,
  `server_id` BIGINT UNSIGNED NOT NULL,
  `baseline_snapshot_id` BIGINT UNSIGNED DEFAULT NULL,
  `current_snapshot_id` BIGINT UNSIGNED NOT NULL,
  `path` VARCHAR(1000) NOT NULL,
  `event_type` VARCHAR(20) NOT NULL COMMENT 'create/delete/modify/chmod/chown/rename',
  `severity` VARCHAR(20) NOT NULL DEFAULT 'high',
  `old_value_json` LONGTEXT DEFAULT NULL,
  `new_value_json` LONGTEXT DEFAULT NULL,
  `occurred_at` DATETIME NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_security_fim_diff_events_policy_server` (`policy_id`, `server_id`),
  KEY `idx_security_fim_diff_events_event_type` (`event_type`),
  KEY `idx_security_fim_diff_events_occurred_at` (`occurred_at`),
  KEY `idx_security_fim_diff_events_path` (`path`(255)),
  CONSTRAINT `fk_security_fim_diff_events_policy`
    FOREIGN KEY (`policy_id`) REFERENCES `security_fim_policies` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巡检差异事件';

CREATE TABLE IF NOT EXISTS `security_fim_alerts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `diff_event_id` BIGINT UNSIGNED NOT NULL,
  `policy_id` BIGINT UNSIGNED NOT NULL,
  `server_id` BIGINT UNSIGNED NOT NULL,
  `title` VARCHAR(255) NOT NULL,
  `summary` VARCHAR(1000) DEFAULT NULL,
  `severity` VARCHAR(20) NOT NULL DEFAULT 'high',
  `status` VARCHAR(20) NOT NULL DEFAULT 'open' COMMENT 'open/acknowledged/resolved/closed',
  `assignee` VARCHAR(100) DEFAULT NULL,
  `first_seen_at` DATETIME NOT NULL,
  `last_seen_at` DATETIME NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_security_fim_alerts_status_severity_created` (`status`, `severity`, `created_at`),
  KEY `idx_security_fim_alerts_server_id` (`server_id`),
  CONSTRAINT `fk_security_fim_alerts_diff_event`
    FOREIGN KEY (`diff_event_id`) REFERENCES `security_fim_diff_events` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巡检完整性告警';
