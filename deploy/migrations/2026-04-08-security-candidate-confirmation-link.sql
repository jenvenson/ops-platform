SET @add_source_vuln_id = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM INFORMATION_SCHEMA.COLUMNS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'security_vulnerabilities'
        AND COLUMN_NAME = 'source_vuln_id'
    ),
    'SELECT 1',
    "ALTER TABLE `security_vulnerabilities` ADD COLUMN `source_vuln_id` BIGINT UNSIGNED NULL COMMENT '派生正式漏洞时关联的候选记录 ID' AFTER `match_mode`"
  )
);
PREPARE stmt FROM @add_source_vuln_id;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @add_confirmed_vuln_id = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM INFORMATION_SCHEMA.COLUMNS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'security_vulnerabilities'
        AND COLUMN_NAME = 'confirmed_vuln_id'
    ),
    'SELECT 1',
    "ALTER TABLE `security_vulnerabilities` ADD COLUMN `confirmed_vuln_id` BIGINT UNSIGNED NULL COMMENT '候选确认后派生的正式漏洞 ID' AFTER `false_positive`"
  )
);
PREPARE stmt FROM @add_confirmed_vuln_id;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @add_source_vuln_id_index = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM INFORMATION_SCHEMA.STATISTICS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'security_vulnerabilities'
        AND INDEX_NAME = 'idx_security_vulns_source_vuln_id'
    ),
    'SELECT 1',
    "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_source_vuln_id` (`source_vuln_id`)"
  )
);
PREPARE stmt FROM @add_source_vuln_id_index;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @add_confirmed_vuln_id_index = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM INFORMATION_SCHEMA.STATISTICS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'security_vulnerabilities'
        AND INDEX_NAME = 'idx_security_vulns_confirmed_vuln_id'
    ),
    'SELECT 1',
    "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_confirmed_vuln_id` (`confirmed_vuln_id`)"
  )
);
PREPARE stmt FROM @add_confirmed_vuln_id_index;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
