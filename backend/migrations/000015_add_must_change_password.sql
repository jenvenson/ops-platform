-- Add must_change_password column to users table (idempotent)
-- 0 = false (password already changed), 1 = true (must change on next login)

-- Only add column if it doesn't already exist
SET @col_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'must_change_password');

SET @sql = IF(@col_exists = 0,
  'ALTER TABLE users ADD COLUMN must_change_password TINYINT(1) NOT NULL DEFAULT 0 AFTER role',
  'SELECT "Column must_change_password already exists" AS msg');

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Existing users (including default admin) should change password on next login
UPDATE users SET must_change_password = 1;
