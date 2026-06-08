-- 创建聚合历史表
CREATE TABLE IF NOT EXISTS aggregated_histories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_name VARCHAR(255) NOT NULL COMMENT '项目名称',
    environment VARCHAR(100) COMMENT '环境',
    status VARCHAR(50) DEFAULT 'pending' COMMENT '状态',
    archive_time DATETIME COMMENT '归档时间',
    download_url TEXT COMMENT '下载地址',
    operator VARCHAR(100) COMMENT '操作人',
    jenkins_job_name VARCHAR(255) COMMENT 'Jenkins任务名称',
    jenkins_build_num INT COMMENT 'Jenkins构建编号',
    jenkins_queue_id BIGINT COMMENT 'Jenkins队列ID',
    jenkins_console_url TEXT COMMENT 'Jenkins控制台日志URL',
    error_message TEXT COMMENT '错误信息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_project_name (project_name),
    INDEX idx_environment (environment),
    INDEX idx_status (status),
    INDEX idx_operator (operator),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='聚合历史记录表';