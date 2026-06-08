-- 安全扫描 Phase 1 数据模型 DDL 草案
-- 说明：
-- 1. 本文件为设计草案，不直接放入 deploy/migrations，避免误执行。
-- 2. 当前目标是为主机/Web 扫描重构建立 run/target/evidence/occurrence 基础设施。

-- ------------------------------------------------------------
-- 1. 扩展 security_scan_runs
-- ------------------------------------------------------------

ALTER TABLE `security_scan_runs`
    ADD COLUMN `run_no` INT NULL COMMENT '同一任务下的运行序号' AFTER `task_id`,
    ADD COLUMN `phase` VARCHAR(30) NULL COMMENT '当前阶段: prepare/discovery/verification/authenticated/reporting/completed' AFTER `status`,
    ADD COLUMN `config_snapshot` JSON NULL COMMENT '本次执行配置快照' AFTER `message`,
    ADD COLUMN `target_snapshot` JSON NULL COMMENT '本次展开目标快照' AFTER `config_snapshot`,
    ADD COLUMN `summary_snapshot` JSON NULL COMMENT '本次统计摘要快照' AFTER `target_snapshot`,
    ADD COLUMN `cancelled_at` DATETIME NULL COMMENT '取消时间' AFTER `completed_at`,
    ADD INDEX `idx_security_scan_runs_run_no` (`task_id`, `run_no`),
    ADD INDEX `idx_security_scan_runs_phase` (`phase`);

-- ------------------------------------------------------------
-- 2. 新增 security_scan_targets
-- ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `security_scan_targets` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `run_id` BIGINT UNSIGNED NOT NULL COMMENT '所属运行 ID',
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '所属任务 ID',
    `parent_target_id` BIGINT UNSIGNED NULL COMMENT '父级目标 ID',
    `target_kind` VARCHAR(20) NOT NULL COMMENT '目标种类: ip/service/url/page/api/form/auth',
    `normalized_target` VARCHAR(500) NOT NULL COMMENT '标准化目标标识',
    `host` VARCHAR(255) NULL COMMENT '主机/IP/域名',
    `port` INT NULL COMMENT '端口',
    `scheme` VARCHAR(20) NULL COMMENT '协议: http/https/tcp/udp',
    `path` VARCHAR(500) NULL COMMENT 'URL 路径或逻辑路径',
    `service_name` VARCHAR(100) NULL COMMENT '服务名称',
    `product_name` VARCHAR(100) NULL COMMENT '产品名称',
    `version` VARCHAR(100) NULL COMMENT '产品版本',
    `status` VARCHAR(20) DEFAULT 'pending' COMMENT 'pending/running/completed/failed/skipped',
    `discovery_source` VARCHAR(30) NULL COMMENT '目标来源: nmap/http/browser/manual',
    `started_at` DATETIME NULL COMMENT '开始时间',
    `completed_at` DATETIME NULL COMMENT '完成时间',
    `metadata_json` JSON NULL COMMENT '扩展元数据',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_scan_targets_run_id` (`run_id`),
    INDEX `idx_scan_targets_task_id` (`task_id`),
    INDEX `idx_scan_targets_parent_id` (`parent_target_id`),
    INDEX `idx_scan_targets_kind_status` (`target_kind`, `status`),
    INDEX `idx_scan_targets_normalized` (`normalized_target`(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='扫描运行目标单元表';

-- ------------------------------------------------------------
-- 3. 新增 security_scan_evidences
-- ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `security_scan_evidences` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `run_id` BIGINT UNSIGNED NOT NULL COMMENT '所属运行 ID',
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '所属任务 ID',
    `target_id` BIGINT UNSIGNED NULL COMMENT '关联目标 ID',
    `evidence_type` VARCHAR(30) NOT NULL COMMENT '证据类型: nmap-service/http-request/http-response/banner/nuclei-result/rule-match/browser-discovery/auth-login/config-check/screenshot',
    `source_engine` VARCHAR(30) NULL COMMENT '来源引擎: nmap/nuclei/browser/rule/auth',
    `digest` CHAR(64) NULL COMMENT '证据摘要，用于去重',
    `request_excerpt` MEDIUMTEXT NULL COMMENT '请求片段',
    `response_excerpt` MEDIUMTEXT NULL COMMENT '响应片段',
    `payload_excerpt` MEDIUMTEXT NULL COMMENT 'payload 片段',
    `metadata_json` JSON NULL COMMENT '证据元数据',
    `raw_json` LONGTEXT NULL COMMENT '原始结构化结果',
    `storage_ref` VARCHAR(500) NULL COMMENT '外部存储引用',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_scan_evidences_run_id` (`run_id`),
    INDEX `idx_scan_evidences_task_id` (`task_id`),
    INDEX `idx_scan_evidences_target_id` (`target_id`),
    INDEX `idx_scan_evidences_type` (`evidence_type`),
    INDEX `idx_scan_evidences_digest` (`digest`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='扫描原始证据表';

-- ------------------------------------------------------------
-- 4. 新增 security_scan_finding_occurrences
-- ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `security_scan_finding_occurrences` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `run_id` BIGINT UNSIGNED NOT NULL COMMENT '所属运行 ID',
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '所属任务 ID',
    `target_id` BIGINT UNSIGNED NULL COMMENT '关联目标 ID',
    `legacy_vulnerability_id` BIGINT UNSIGNED NULL COMMENT '兼容当前 security_vulnerabilities 主键',
    `finding_key` VARCHAR(255) NULL COMMENT '跨运行稳定识别键',
    `finding_family` VARCHAR(20) NULL COMMENT '结果家族: vulnerability/inventory',
    `finding_source` VARCHAR(30) NULL COMMENT '结果来源',
    `severity` VARCHAR(20) NULL COMMENT '风险等级',
    `confidence` VARCHAR(20) NULL COMMENT '置信度',
    `match_mode` VARCHAR(30) NULL COMMENT '匹配方式',
    `primary_cve_id` VARCHAR(50) NULL COMMENT '主 CVE',
    `vuln_db_id` BIGINT UNSIGNED NULL COMMENT '关联漏洞库主记录 ID',
    `title` VARCHAR(200) NULL COMMENT '标题快照',
    `status` VARCHAR(20) DEFAULT 'open' COMMENT '处置状态',
    `verification_status` VARCHAR(20) DEFAULT 'pending' COMMENT '验证状态',
    `evidence_count` INT DEFAULT 0 COMMENT '关联证据数',
    `first_seen_at` DATETIME NULL COMMENT '本次运行首见时间',
    `last_seen_at` DATETIME NULL COMMENT '本次运行末次命中时间',
    `metadata_json` JSON NULL COMMENT '扩展元数据',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_scan_occurrences_run_id` (`run_id`),
    INDEX `idx_scan_occurrences_task_id` (`task_id`),
    INDEX `idx_scan_occurrences_target_id` (`target_id`),
    INDEX `idx_scan_occurrences_legacy_vuln_id` (`legacy_vulnerability_id`),
    INDEX `idx_scan_occurrences_source` (`finding_source`),
    INDEX `idx_scan_occurrences_status` (`status`),
    INDEX `idx_scan_occurrences_verification` (`verification_status`),
    INDEX `idx_scan_occurrences_primary_cve` (`primary_cve_id`),
    INDEX `idx_scan_occurrences_finding_key` (`finding_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='扫描运行命中记录表';

-- ------------------------------------------------------------
-- 5. 可选桥接表（后续实现时再决定是否需要）
-- ------------------------------------------------------------
-- CREATE TABLE `security_scan_occurrence_evidences` (...)
-- CREATE TABLE `security_scan_occurrence_labels` (...)
