-- 漏洞结果语义字段：来源、家族、置信度、主 CVE、漏洞库关联
-- 2026-04-03
-- 幂等执行，避免开发环境重复启动时因重复字段/索引失败

SET @schema_name = DATABASE();

SET @sql = IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'security_vulnerabilities'
      AND COLUMN_NAME = 'finding_source'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `finding_source` VARCHAR(30) NULL COMMENT '结果来源: web-template, web-rule, host-template, host-version-match, asset-inventory' AFTER `vuln_url`"
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
      AND COLUMN_NAME = 'finding_family'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `finding_family` VARCHAR(20) NULL COMMENT '结果家族: vulnerability, inventory' AFTER `finding_source`"
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
      AND COLUMN_NAME = 'confidence'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `confidence` VARCHAR(20) NULL COMMENT '结果置信度: high, medium, low' AFTER `finding_family`"
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
      AND COLUMN_NAME = 'primary_cve_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `primary_cve_id` VARCHAR(50) NULL COMMENT '主 CVE 编号' AFTER `confidence`"
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
      AND COLUMN_NAME = 'vuln_db_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `vuln_db_id` BIGINT UNSIGNED NULL COMMENT '关联漏洞库主记录 ID' AFTER `primary_cve_id`"
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
      AND COLUMN_NAME = 'match_mode'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD COLUMN `match_mode` VARCHAR(30) NULL COMMENT '匹配模式: template, rule, version-range, fuzzy-product, inventory' AFTER `vuln_db_id`"
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
      AND INDEX_NAME = 'idx_security_vulns_finding_source'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_finding_source` (`finding_source`)"
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
      AND INDEX_NAME = 'idx_security_vulns_finding_family'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_finding_family` (`finding_family`)"
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
      AND INDEX_NAME = 'idx_security_vulns_confidence'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_confidence` (`confidence`)"
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
      AND INDEX_NAME = 'idx_security_vulns_vuln_db_id'
  ),
  'SELECT 1',
  "ALTER TABLE `security_vulnerabilities` ADD INDEX `idx_security_vulns_vuln_db_id` (`vuln_db_id`)"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
