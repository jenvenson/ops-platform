-- 部署记录表迁移
USE ops_platform;

-- 部署记录表
CREATE TABLE IF NOT EXISTS deploy_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    app_name VARCHAR(100) NOT NULL COMMENT '应用名称',
    env_id BIGINT UNSIGNED NOT NULL COMMENT '环境ID',
    env_name VARCHAR(50) NOT NULL COMMENT '环境名称',
    env_type VARCHAR(10) NOT NULL COMMENT '环境类型: dev/test/prod',
    project_code VARCHAR(50) NOT NULL COMMENT '项目代码',
    deploy_type VARCHAR(20) NOT NULL COMMENT '部署类型: frontend/backend/all',
    jenkins_job VARCHAR(200) COMMENT 'Jenkins任务名',
    jenkins_build_num INT DEFAULT 0 COMMENT 'Jenkins构建号',
    jenkins_queue_id BIGINT DEFAULT 0 COMMENT 'Jenkins队列ID',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '状态: pending/running/success/failed',
    error_message TEXT COMMENT '错误信息',
    start_time DATETIME COMMENT '开始时间',
    end_time DATETIME COMMENT '结束时间',
    duration INT DEFAULT 0 COMMENT '持续时间(秒)',
    triggered_by VARCHAR(100) COMMENT '触发人',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_app_id (app_id),
    INDEX idx_env_id (env_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部署记录表';
