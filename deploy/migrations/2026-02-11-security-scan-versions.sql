-- 添加 nuclei 版本和模板版本字段
ALTER TABLE security_scan_tasks
ADD COLUMN nuclei_version VARCHAR(50) AFTER message,
ADD COLUMN template_version VARCHAR(50) AFTER nuclei_version;
