-- 添加操作人字段到归档记录表
ALTER TABLE archive_records ADD COLUMN operator VARCHAR(100);
CREATE INDEX idx_archive_records_operator ON archive_records(operator);
