-- 创建漏洞知识库表
-- 执行: mysql -u root -p ops_platform < migrations/xxxx-xx-xx-vuln-db.sql

CREATE TABLE IF NOT EXISTS vulnerability_database (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    cve_id VARCHAR(50) NOT NULL COMMENT 'CVE 编号',
    cnvd_id VARCHAR(50) COMMENT 'CNVD 编号',
    cnnvd_id VARCHAR(50) COMMENT 'CNNVD 编号',
    cncve_id VARCHAR(50) COMMENT 'CNCVE 编号',

    title VARCHAR(200) NOT NULL COMMENT '漏洞标题',
    description TEXT COMMENT '漏洞描述',
    vuln_type VARCHAR(50) COMMENT '漏洞类型: rce, xss, sqli, etc.',
    severity VARCHAR(20) COMMENT '严重程度: critical, high, medium, low',
    cvss_score DECIMAL(3,1) COMMENT 'CVSS 分数',
    cvss_vector VARCHAR(200) COMMENT 'CVSS 向量字符串',

    affected_product VARCHAR(200) COMMENT '受影响产品',
    affected_version VARCHAR(200) COMMENT '受影响版本',

    solution TEXT COMMENT '修复建议',
    patch_url VARCHAR(500) COMMENT '补丁链接',
    workaround TEXT COMMENT '缓解措施',

    references TEXT COMMENT '参考链接（逗号分隔）',
    cwe_id VARCHAR(20) COMMENT 'CWE 编号',

    tags VARCHAR(200) COMMENT '标签（逗号分隔）',
    source VARCHAR(50) COMMENT '数据来源: nvd, cnnvd, cnvd, manual',
    last_updated DATETIME COMMENT '最后更新时间',

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY idx_cve_id (cve_id),
    INDEX idx_cnvd_id (cnvd_id),
    INDEX idx_cnnvd_id (cnnvd_id),
    INDEX idx_severity (severity),
    INDEX idx_vuln_type (vuln_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='漏洞知识库';
