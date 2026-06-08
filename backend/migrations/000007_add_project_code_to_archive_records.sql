-- 添加 project_code 字段到 archive_records 表
USE ops_platform;

ALTER TABLE archive_records
ADD COLUMN project_code VARCHAR(50) NOT NULL DEFAULT 'unknown' COMMENT '项目编号，用于下载地址' AFTER deploy_type;

-- 更新现有记录的 project_code
UPDATE archive_records ar
INNER JOIN applications app ON ar.app_id = app.id
INNER JOIN projects p ON app.project_id = p.id
SET ar.project_code = p.code
WHERE ar.project_code = 'unknown' OR ar.project_code = '';
