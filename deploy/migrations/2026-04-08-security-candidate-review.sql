ALTER TABLE `security_vulnerabilities`
    ADD COLUMN `review_status` VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '候选复核状态: pending, needs-test, confirmed, rejected' AFTER `false_positive`,
    ADD COLUMN `review_note` TEXT NULL COMMENT '候选复核备注' AFTER `review_status`,
    ADD COLUMN `reviewed_by` BIGINT UNSIGNED NULL COMMENT '候选复核人ID' AFTER `review_note`,
    ADD COLUMN `reviewed_at` DATETIME NULL COMMENT '候选复核时间' AFTER `reviewed_by`,
    ADD INDEX `idx_review_status` (`review_status`),
    ADD INDEX `idx_reviewed_by` (`reviewed_by`);
