-- Jenkins API 集成数据库迁移
-- 添加 Jenkins 相关字段到 pipelines 和 pipeline_executions 表

-- =====================================================
-- 1. 扩展 pipelines 表，添加 Jenkins 配置字段
-- =====================================================
ALTER TABLE pipelines 
ADD COLUMN jenkins_url VARCHAR(255) COMMENT 'Jenkins 地址（可选，覆盖全局配置）',
ADD COLUMN jenkins_view VARCHAR(100) COMMENT 'Jenkins View 名称',
ADD COLUMN jenkins_job VARCHAR(100) COMMENT 'Jenkins Job 名称';

-- =====================================================
-- 2. 扩展 pipeline_executions 表，添加 Jenkins 构建信息
-- =====================================================
ALTER TABLE pipeline_executions
ADD COLUMN jenkins_build_number INT COMMENT 'Jenkins 构建号',
ADD COLUMN jenkins_build_url VARCHAR(255) COMMENT 'Jenkins 构建链接',
ADD COLUMN jenkins_queue_id BIGINT COMMENT 'Jenkins 队列 ID';

-- =====================================================
-- 3. 添加索引优化查询
-- =====================================================
CREATE INDEX idx_pipeline_executions_jenkins_build ON pipeline_executions(jenkins_build_number);
