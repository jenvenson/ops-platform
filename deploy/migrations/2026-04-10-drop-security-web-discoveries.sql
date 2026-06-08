-- 删除旧版 Web 发现结果兼容表
-- 2026-04-10
-- 前提：
-- 1. 前端与导出已切到 security_scan_targets / evidences / occurrences
-- 2. 后端已移除 /tasks/:id/discoveries 接口与旧 dual-write

SET @schema_name = DATABASE();

SET @sql = IF(
  EXISTS(
    SELECT 1
    FROM information_schema.TABLES
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'security_web_discoveries'
  ),
  'DROP TABLE `security_web_discoveries`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
