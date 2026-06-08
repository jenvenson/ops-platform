-- 扫描进度功能字段添加
-- 创建时间: 2026-02-11

-- 为安全扫描任务表添加进度相关字段
ALTER TABLE `security_scan_tasks`
ADD COLUMN `progress` INT DEFAULT 0 COMMENT '进度百分比 0-100' AFTER `status`,
ADD COLUMN `total_ips` INT DEFAULT 0 COMMENT '总共需要扫描的 IP 数量' AFTER `progress`,
ADD COLUMN `scanned_ips` INT DEFAULT 0 COMMENT '已扫描的 IP 数量' AFTER `total_ips`,
ADD COLUMN `message` VARCHAR(255) DEFAULT NULL COMMENT '当前状态信息' AFTER `scanned_ips`;
