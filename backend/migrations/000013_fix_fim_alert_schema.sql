SET @table_exists = (
    SELECT COUNT(*)
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'security_fim_alerts'
);

SET @sql = IF(
    @table_exists = 0,
    'SELECT 1',
    IF(
        (
            SELECT COUNT(*)
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'security_fim_alerts'
              AND column_name = 'path'
        ) = 0,
        'ALTER TABLE `security_fim_alerts` ADD COLUMN `path` VARCHAR(1000) DEFAULT NULL AFTER `server_id`',
        'SELECT 1'
    )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
    @table_exists = 0,
    'SELECT 1',
    IF(
        (
            SELECT COUNT(*)
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'security_fim_alerts'
              AND column_name = 'event_type'
        ) = 0,
        'ALTER TABLE `security_fim_alerts` ADD COLUMN `event_type` VARCHAR(20) DEFAULT NULL AFTER `path`',
        'SELECT 1'
    )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
    @table_exists = 0,
    'SELECT 1',
    IF(
        (
            SELECT COUNT(*)
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'security_fim_alerts'
              AND column_name = 'occurrence_count'
        ) = 0,
        'ALTER TABLE `security_fim_alerts` ADD COLUMN `occurrence_count` BIGINT NOT NULL DEFAULT 1 AFTER `status`',
        'SELECT 1'
    )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @path_index_needs_reset = IF(
    @table_exists = 0,
    0,
    (
        SELECT COUNT(*)
        FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = 'security_fim_alerts'
          AND index_name = 'idx_security_fim_alerts_path'
          AND (sub_part IS NULL OR sub_part <> 255)
    )
);

SET @sql = IF(
    @path_index_needs_reset > 0,
    'ALTER TABLE `security_fim_alerts` DROP INDEX `idx_security_fim_alerts_path`',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
    @table_exists = 0,
    'SELECT 1',
    IF(
        (
            SELECT COUNT(*)
            FROM information_schema.statistics
            WHERE table_schema = DATABASE()
              AND table_name = 'security_fim_alerts'
              AND index_name = 'idx_security_fim_alerts_path'
              AND column_name = 'path'
              AND sub_part = 255
        ) = 0,
        'ALTER TABLE `security_fim_alerts` ADD INDEX `idx_security_fim_alerts_path` (`path`(255))',
        'SELECT 1'
    )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
    @table_exists = 0,
    'SELECT 1',
    IF(
        (
            SELECT COUNT(*)
            FROM information_schema.statistics
            WHERE table_schema = DATABASE()
              AND table_name = 'security_fim_alerts'
              AND index_name = 'idx_security_fim_alerts_event_type'
        ) = 0,
        'ALTER TABLE `security_fim_alerts` ADD INDEX `idx_security_fim_alerts_event_type` (`event_type`)',
        'SELECT 1'
    )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
    @table_exists = 0,
    'SELECT 1',
    IF(
        (
            SELECT COUNT(*)
            FROM information_schema.statistics
            WHERE table_schema = DATABASE()
              AND table_name = 'security_fim_alerts'
              AND index_name = 'idx_security_fim_alerts_status_severity_created'
        ) = 0,
        'ALTER TABLE `security_fim_alerts` ADD INDEX `idx_security_fim_alerts_status_severity_created` (`status`, `severity`, `created_at`)',
        'SELECT 1'
    )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
