-- 安全扫描模块：扩展 target_type 字段支持 URL 类型
-- 2026-02-12

ALTER TABLE `security_scan_tasks`
MODIFY COLUMN `target_type` VARCHAR(20) NOT NULL COMMENT '目标类型: cidr-网段, ip_list-IP列表, url-URL地址';
