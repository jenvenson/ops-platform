-- 安全扫描模块数据库表
-- 创建时间: 2026-02-08

-- 安全扫描任务表
CREATE TABLE IF NOT EXISTS `security_scan_tasks` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(100) NOT NULL COMMENT '任务名称',
    `target_type` VARCHAR(20) NOT NULL COMMENT '目标类型: cidr-网段, ip_list-IP列表',
    `target` VARCHAR(500) NOT NULL COMMENT '扫描目标',
    `status` VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending-等待中, running-运行中, completed-已完成, failed-失败',
    `high_risk` INT DEFAULT 0 COMMENT '高危漏洞数量',
    `medium_risk` INT DEFAULT 0 COMMENT '中危漏洞数量',
    `low_risk` INT DEFAULT 0 COMMENT '低危漏洞数量',
    `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
    `completed_at` DATETIME DEFAULT NULL COMMENT '完成时间',
    `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建人ID',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_status` (`status`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='安全扫描任务表';

-- 安全资产表（发现的端口和服务）
CREATE TABLE IF NOT EXISTS `security_assets` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '关联任务ID',
    `ip` VARCHAR(45) DEFAULT NULL COMMENT 'IP地址',
    `port` INT DEFAULT NULL COMMENT '端口号',
    `protocol` VARCHAR(10) DEFAULT NULL COMMENT '协议: tcp/udp',
    `service_name` VARCHAR(50) DEFAULT NULL COMMENT '服务名称',
    `version` VARCHAR(100) DEFAULT NULL COMMENT '版本号',
    `os_info` VARCHAR(100) DEFAULT NULL COMMENT '操作系统信息',
    `banner` TEXT DEFAULT NULL COMMENT '服务Banner信息',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_ip` (`ip`),
    INDEX `idx_port` (`port`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='安全资产表';

-- 安全漏洞表
CREATE TABLE IF NOT EXISTS `security_vulnerabilities` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '关联任务ID',
    `asset_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联资产ID',
    `ip` VARCHAR(45) DEFAULT NULL COMMENT '目标IP',
    `port` INT DEFAULT NULL COMMENT '目标端口',
    `severity` VARCHAR(20) NOT NULL COMMENT '严重程度: high-高危, medium-中危, low-低危, info-信息',
    `cve_id` VARCHAR(50) DEFAULT NULL COMMENT 'CVE编号',
    `cve_type` VARCHAR(50) DEFAULT NULL COMMENT '漏洞类型',
    `title` VARCHAR(200) DEFAULT NULL COMMENT '漏洞标题',
    `description` TEXT DEFAULT NULL COMMENT '漏洞描述',
    `solution` TEXT DEFAULT NULL COMMENT '修复方案',
    `cvss_score` DOUBLE DEFAULT NULL COMMENT 'CVSS评分',
    `cvss_vector` VARCHAR(200) DEFAULT NULL COMMENT 'CVSS向量',
    `payload` TEXT DEFAULT NULL COMMENT '探测使用的Payload',
    `request` TEXT DEFAULT NULL COMMENT '请求片段',
    `response` TEXT DEFAULT NULL COMMENT '响应片段',
    `reference_url` VARCHAR(500) DEFAULT NULL COMMENT '参考链接',
    `status` VARCHAR(20) DEFAULT 'open' COMMENT '状态: open-待处理, acknowledged-已确认, fixed-已修复, ignored-已忽略',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_asset_id` (`asset_id`),
    INDEX `idx_ip` (`ip`),
    INDEX `idx_severity` (`severity`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='安全漏洞表';

-- 安全扫描菜单配置
INSERT INTO `menus` (`key`, `path`, `title`, `icon`, `parent_id`, `sort`, `status`) VALUES
('security', '/security', '安全管理', 'SafetyOutlined', 0, 4, 1),
('security-overview', '/security/overview', '安全概览', 'DashboardOutlined', (SELECT id FROM menus WHERE `key` = 'security'), 1, 1),
('security-tasks', '/security/tasks', '扫描任务', 'ScanOutlined', (SELECT id FROM menus WHERE `key` = 'security'), 2, 1),
('security-vulnerabilities', '/security/vulnerabilities', '漏洞管理', 'BugOutlined', (SELECT id FROM menus WHERE `key` = 'security'), 3, 1)
ON DUPLICATE KEY UPDATE title = VALUES(title);
