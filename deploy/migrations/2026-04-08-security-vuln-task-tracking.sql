-- 漏洞归属跟踪：保留首次发现任务与最近一次命中任务
-- 2026-04-08
-- 幂等执行，兼容开发环境重复启动

SET @schema_name = DATABASE();

SET @sql = IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'security_vulnerabilities'
      AND COLUMN_NAME = 'first_task_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `first_task_id` BIGINT UNSIGNED NULL COMMENT '首次发现该漏洞的任务 ID' AFTER `task_id`"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'security_vulnerabilities'
      AND COLUMN_NAME = 'last_task_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `last_task_id` BIGINT UNSIGNED NULL COMMENT '最近一次命中该漏洞的任务 ID' AFTER `first_task_id`"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  EXISTS(
    SELECT 1
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'security_vulnerabilities'
      AND INDEX_NAME = 'idx_security_vulns_first_task_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_first_task_id` (`first_task_id`)"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  EXISTS(
    SELECT 1
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'security_vulnerabilities'
      AND INDEX_NAME = 'idx_security_vulns_last_task_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_last_task_id` (`last_task_id`)"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `security_vulnerabilities`
SET `first_task_id` = `task_id`
WHERE `first_task_id` IS NULL
  AND `task_id` IS NOT NULL;

UPDATE `security_vulnerabilities`
SET `last_task_id` = `task_id`
WHERE `last_task_id` IS NULL
  AND `task_id` IS NOT NULL;
