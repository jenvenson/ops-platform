-- 安全管理模块数据库表结构
-- 创建时间: 2026-02-08

-- 安全扫描任务表
CREATE TABLE IF NOT EXISTS `security_scan_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(255) NOT NULL COMMENT '任务名称',
  `target_type` ENUM('cidr', 'ip_list') NOT NULL COMMENT '目标类型: cidr网段, ip_listIP列表',
  `target` VARCHAR(1024) NOT NULL COMMENT '扫描目标',
  `status` ENUM('pending', 'running', 'completed', 'failed') NOT NULL DEFAULT 'pending' COMMENT '任务状态',
  `high_risk` INT NOT NULL DEFAULT 0 COMMENT '高危漏洞数量',
  `medium_risk` INT NOT NULL DEFAULT 0 COMMENT '中危漏洞数量',
  `low_risk` INT NOT NULL DEFAULT 0 COMMENT '低危漏洞数量',
  `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
  `completed_at` DATETIME DEFAULT NULL COMMENT '完成时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  INDEX `idx_status` (`status`),
  INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='安全扫描任务表';

-- 安全资产表
CREATE TABLE IF NOT EXISTS `security_assets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `task_id` BIGINT UNSIGNED NOT NULL COMMENT '所属任务ID',
  `ip` VARCHAR(45) NOT NULL COMMENT 'IP地址',
  `port` INT NOT NULL DEFAULT 0 COMMENT '端口号',
  `protocol` VARCHAR(16) NOT NULL DEFAULT 'tcp' COMMENT '协议',
  `service_name` VARCHAR(128) DEFAULT NULL COMMENT '服务名称',
  `version` VARCHAR(256) DEFAULT NULL COMMENT '版本信息',
  `os_info` VARCHAR(256) DEFAULT NULL COMMENT '操作系统信息',
  `banner` TEXT DEFAULT NULL COMMENT '服务Banner',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX `idx_task_id` (`task_id`),
  INDEX `idx_ip_port` (`ip`, `port`),
  CONSTRAINT `fk_security_asset_task` FOREIGN KEY (`task_id`) REFERENCES `security_scan_tasks` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='安全资产表';

-- 安全漏洞表
CREATE TABLE IF NOT EXISTS `security_vulnerabilities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `task_id` BIGINT UNSIGNED NOT NULL COMMENT '所属任务ID',
  `asset_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联资产ID',
  `ip` VARCHAR(45) NOT NULL COMMENT '目标IP',
  `port` INT NOT NULL DEFAULT 0 COMMENT '目标端口',
  `severity` ENUM('high', 'medium', 'low', 'info') NOT NULL DEFAULT 'medium' COMMENT '严重程度',
  `cve_id` VARCHAR(32) DEFAULT NULL COMMENT 'CVE编号',
  `cve_type` VARCHAR(64) DEFAULT NULL COMMENT 'CVE类型',
  `title` VARCHAR(512) NOT NULL COMMENT '漏洞标题',
  `description` TEXT DEFAULT NULL COMMENT '漏洞描述',
  `solution` TEXT DEFAULT NULL COMMENT '修复方案',
  `cvss_score` DECIMAL(3,1) NOT NULL DEFAULT 0.0 COMMENT 'CVSS评分',
  `cvss_vector` VARCHAR(128) DEFAULT NULL COMMENT 'CVSS向量',
  `payload` TEXT DEFAULT NULL COMMENT '测试Payload',
  `request` TEXT DEFAULT NULL COMMENT '请求片段',
  `response` TEXT DEFAULT NULL COMMENT '响应片段',
  `reference_url` VARCHAR(1024) DEFAULT NULL COMMENT '参考链接',
  `status` ENUM('open', 'acknowledged', 'fixed', 'ignored') NOT NULL DEFAULT 'open' COMMENT '漏洞状态',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发现时间',
  INDEX `idx_task_id` (`task_id`),
  INDEX `idx_severity` (`severity`),
  INDEX `idx_status` (`status`),
  INDEX `idx_cve_id` (`cve_id`),
  CONSTRAINT `fk_vuln_task` FOREIGN KEY (`task_id`) REFERENCES `security_scan_tasks` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_vuln_asset` FOREIGN KEY (`asset_id`) REFERENCES `security_assets` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='安全漏洞表';
