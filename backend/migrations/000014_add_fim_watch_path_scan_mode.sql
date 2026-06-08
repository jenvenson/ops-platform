SET @has_scan_mode := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'security_fim_watch_paths'
    AND COLUMN_NAME = 'scan_mode'
);

SET @ddl := IF(
  @has_scan_mode = 0,
  'ALTER TABLE security_fim_watch_paths ADD COLUMN scan_mode VARCHAR(20) NOT NULL DEFAULT ''full_hash'' AFTER path',
  'SELECT 1'
);

PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE security_fim_watch_paths
SET scan_mode = 'full_hash'
WHERE scan_mode IS NULL OR scan_mode = '';
