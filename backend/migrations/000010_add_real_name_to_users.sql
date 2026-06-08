-- 添加 real_name 字段到用户表
ALTER TABLE users ADD COLUMN real_name VARCHAR(50) DEFAULT NULL COMMENT '真实姓名';