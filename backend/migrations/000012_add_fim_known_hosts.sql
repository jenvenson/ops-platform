-- FIM已知主机密钥管理表
-- 用于存储FIM扫描时信任的服务器SSH主机密钥

-- 主表：已知主机密钥
CREATE TABLE IF NOT EXISTS fim_known_hosts (
    -- 主键和基本信息
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    
    -- 主机标识
    hostname VARCHAR(255) NOT NULL COMMENT '主机名或IP地址',
    port INT NOT NULL DEFAULT 22 COMMENT 'SSH端口',
    
    -- 密钥信息
    key_type VARCHAR(50) NOT NULL COMMENT '密钥类型：ssh-rsa, ecdsa-sha2-nistp256, ssh-ed25519',
    public_key TEXT NOT NULL COMMENT '完整的公钥内容（OpenSSH格式）',
    fingerprint_sha256 VARCHAR(64) NOT NULL COMMENT 'SHA256指纹（用于显示和验证）',
    
    -- 关联信息
    server_id BIGINT UNSIGNED COMMENT '关联的服务器ID（可选，如果服务器在CMDB中）',
    description VARCHAR(500) COMMENT '主机描述/备注',
    tags JSON COMMENT '标签：["production", "critical"]',
    
    -- 验证状态
    verification_status ENUM('unverified', 'verified', 'expired') DEFAULT 'verified' COMMENT '验证状态',
    verified_by VARCHAR(100) COMMENT '验证人',
    verified_at DATETIME COMMENT '验证时间',
    
    -- 使用统计
    last_used_at DATETIME COMMENT '最后使用时间',
    use_count INT DEFAULT 0 COMMENT '使用次数',
    
    -- 审计字段
    added_by VARCHAR(100) NOT NULL COMMENT '添加人',
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '添加时间',
    updated_by VARCHAR(100) COMMENT '最后更新人',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    -- 状态
    is_enabled BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    
    -- 索引
    UNIQUE KEY uk_host_port_keytype (hostname, port, key_type),
    INDEX idx_hostname (hostname),
    INDEX idx_server_id (server_id),
    INDEX idx_verification_status (verification_status),
    INDEX idx_is_enabled (is_enabled),
    INDEX idx_last_used_at (last_used_at)
    
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci 
COMMENT='FIM已知主机密钥白名单';

-- 辅助表：密钥变更历史
CREATE TABLE IF NOT EXISTS fim_known_hosts_history (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    host_id BIGINT UNSIGNED NOT NULL COMMENT '关联的主机ID',
    
    -- 变更类型
    action ENUM('added', 'updated', 'deleted', 'key_changed') NOT NULL COMMENT '操作类型',
    
    -- 快照
    old_key_type VARCHAR(50) COMMENT '旧密钥类型',
    old_public_key TEXT COMMENT '旧公钥',
    old_fingerprint VARCHAR(64) COMMENT '旧指纹',
    
    new_key_type VARCHAR(50) COMMENT '新密钥类型',
    new_public_key TEXT COMMENT '新公钥',
    new_fingerprint VARCHAR(64) COMMENT '新指纹',
    
    -- 审计
    operated_by VARCHAR(100) NOT NULL COMMENT '操作人',
    operated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
    reason TEXT COMMENT '变更原因/备注',
    ip_address VARCHAR(45) COMMENT '操作IP',
    
    INDEX idx_host_id (host_id),
    INDEX idx_operated_at (operated_at)
    
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='主机密钥变更历史';

-- 辅助表：连接尝试日志
CREATE TABLE IF NOT EXISTS fim_ssh_connection_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    
    -- 连接信息
    hostname VARCHAR(255) NOT NULL COMMENT '尝试连接的主机',
    port INT NOT NULL COMMENT '端口',
    
    -- 结果
    result ENUM('success', 'key_not_found', 'key_mismatch', 'connection_failed') NOT NULL COMMENT '连接结果',
    error_message TEXT COMMENT '错误信息',
    
    -- 密钥信息（如果提供）
    presented_key_type VARCHAR(50) COMMENT '服务器提供的密钥类型',
    presented_fingerprint VARCHAR(64) COMMENT '服务器提供的密钥指纹',
    expected_fingerprint VARCHAR(64) COMMENT '期望的密钥指纹',
    
    -- 审计
    attempted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '尝试时间',
    server_id BIGINT UNSIGNED COMMENT '关联的服务器ID',
    policy_id BIGINT UNSIGNED COMMENT '关联的策略ID',
    snapshot_id BIGINT UNSIGNED COMMENT '关联的快照ID',
    
    INDEX idx_hostname_port (hostname, port),
    INDEX idx_result (result),
    INDEX idx_attempted_at (attempted_at),
    INDEX idx_server_id (server_id)
    
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='FIM SSH连接尝试日志';
