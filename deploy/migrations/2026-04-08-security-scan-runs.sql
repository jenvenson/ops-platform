ALTER TABLE `security_scan_tasks`
    ADD COLUMN `current_run_id` BIGINT UNSIGNED NULL COMMENT '当前执行记录ID' AFTER `low_risk`,
    ADD COLUMN `latest_run_id` BIGINT UNSIGNED NULL COMMENT '最近一次执行记录ID' AFTER `current_run_id`,
    ADD INDEX `idx_current_run_id` (`current_run_id`),
    ADD INDEX `idx_latest_run_id` (`latest_run_id`);

CREATE TABLE IF NOT EXISTS `security_scan_runs` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '所属任务ID',
    `task_name` VARCHAR(100) NOT NULL COMMENT '任务名称快照',
    `target_type` VARCHAR(20) NOT NULL COMMENT '目标类型快照',
    `target` VARCHAR(500) NOT NULL COMMENT '目标快照',
    `scan_type` VARCHAR(20) NOT NULL COMMENT '扫描类型快照',
    `status` VARCHAR(20) DEFAULT 'pending' COMMENT '执行状态',
    `progress` INT DEFAULT 0 COMMENT '执行进度 0-100',
    `total_targets` INT DEFAULT 0 COMMENT '总目标数',
    `scanned_targets` INT DEFAULT 0 COMMENT '已扫描目标数',
    `message` VARCHAR(255) DEFAULT NULL COMMENT '当前状态信息',
    `high_risk` INT DEFAULT 0 COMMENT '高危数量',
    `medium_risk` INT DEFAULT 0 COMMENT '中危数量',
    `low_risk` INT DEFAULT 0 COMMENT '低危数量',
    `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
    `completed_at` DATETIME DEFAULT NULL COMMENT '完成时间',
    `triggered_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '触发人ID',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='安全扫描执行记录表';
