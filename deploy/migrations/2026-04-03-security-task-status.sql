-- 安全扫描任务：补齐运行状态枚举，支持暂停与取消
-- 2026-04-03

ALTER TABLE `security_scan_tasks`
MODIFY COLUMN `status` ENUM('pending', 'running', 'paused', 'cancelled', 'completed', 'failed')
NOT NULL DEFAULT 'pending'
COMMENT '任务状态: pending-待执行, running-执行中, paused-已暂停, cancelled-已取消, completed-已完成, failed-失败';
