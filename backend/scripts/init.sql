-- backend/scripts/init.sql
CREATE DATABASE IF NOT EXISTS ops_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE ops_platform;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    real_name VARCHAR(50) COMMENT '姓名',
    email VARCHAR(100),
    role ENUM('admin', 'user') NOT NULL DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_username (username),
    KEY idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 插入默认管理员用户 (密码: admin123)
-- bcrypt hash for "admin123": $2a$10$csSTUM5UQrtt8E/rQ5dZlOk.MQvjEqFRUPxNkTarOLxalC3pBfZny
INSERT INTO users (username, password, real_name, email, role)
VALUES ('admin', '$2a$10$csSTUM5UQrtt8E/rQ5dZlOk.MQvjEqFRUPxNkTarOLxalC3pBfZny', '管理员', 'admin@example.com', 'admin')
ON DUPLICATE KEY UPDATE username=username;

-- 角色表
CREATE TABLE IF NOT EXISTS roles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL COMMENT '角色名称',
    code VARCHAR(50) NOT NULL UNIQUE COMMENT '角色编码',
    description VARCHAR(255) COMMENT '角色描述',
    status TINYINT DEFAULT 1 COMMENT '状态: 1-启用, 0-禁用',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_code (code),
    KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色表';

-- 用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    role_id BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_role (user_id, role_id),
    KEY idx_role_id (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表';

-- 更新用户表，将 role 字段改为存储角色 code
UPDATE users SET role = 'admin' WHERE username = 'admin';
UPDATE users SET role = 'user' WHERE username = 'zhangsan';

-- 注意：角色数据请通过后端 API 或管理界面初始化
-- 默认角色 SQL（使用 utf8mb4 编码执行）：
-- INSERT INTO roles (name, code, description, status) VALUES
-- ('超级管理员', 'admin', '拥有所有权限', 1),
-- ('运维人员', 'ops', '负责系统运维工作', 1),
-- ('开发人员', 'dev', '负责应用开发', 1),
-- ('普通用户', 'user', '普通用户角色', 1);
