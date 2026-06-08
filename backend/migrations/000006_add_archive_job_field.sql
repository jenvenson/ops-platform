-- 添加 Jenkins 归档流水线字段
USE ops_platform;

-- 为 applications 表添加 jenkins_archive_job 字段
ALTER TABLE applications ADD COLUMN jenkins_archive_job VARCHAR(200) COMMENT 'Jenkins归档流水线名称' AFTER jenkins_job;
