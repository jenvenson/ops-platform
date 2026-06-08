-- Migration: Add scan_type to security_scan_tasks
-- Date: 2026-02-11

-- 添加扫描类型字段
ALTER TABLE security_scan_tasks
ADD COLUMN scan_type VARCHAR(20) DEFAULT 'all' COMMENT '扫描类型: host=主机漏洞, web=Web漏洞, all=全面扫描'
AFTER target;

-- 更新现有记录为全面扫描
UPDATE security_scan_tasks SET scan_type = 'all' WHERE scan_type IS NULL;
