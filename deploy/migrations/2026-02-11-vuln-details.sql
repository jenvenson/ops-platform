-- 添加漏洞详细报告所需字段
-- 执行: mysql -u root -p ops_platform < migrations/xxxx-xx-xx-vuln-details.sql

ALTER TABLE security_vulnerabilities
    ADD COLUMN IF NOT EXISTS protocol VARCHAR(10) DEFAULT 'TCP' COMMENT '协议' AFTER port,

    -- CVSS 信息
    ADD COLUMN IF NOT EXISTS cvss_vector VARCHAR(200) COMMENT 'CVSS 向量字符串' AFTER cvss_score,

    -- 漏洞标识
    ADD COLUMN IF NOT EXISTS cnvd_id VARCHAR(50) COMMENT 'CNVD 编号' AFTER cve_id,
    ADD COLUMN IF NOT EXISTS cnnvd_id VARCHAR(50) COMMENT 'CNNVD 编号' AFTER cnvd_id,
    ADD COLUMN IF NOT EXISTS cncve_id VARCHAR(50) COMMENT 'CNCVE 编号' AFTER cnnvd_id,

    -- 漏洞信息
    ADD COLUMN IF NOT EXISTS vuln_type VARCHAR(50) COMMENT '漏洞类型: rce, xss, sqli, etc.' AFTER description,
    ADD COLUMN IF NOT EXISTS solution TEXT COMMENT '修复建议' AFTER vuln_type,

    -- 扫描信息
    ADD COLUMN IF NOT EXISTS scanner VARCHAR(20) DEFAULT 'nuclei' COMMENT '扫描引擎',
    ADD COLUMN IF NOT EXISTS template_id VARCHAR(100) COMMENT 'Nuclei 模板ID',
    ADD COLUMN IF NOT EXISTS scan_method VARCHAR(50) DEFAULT '非授权扫描' COMMENT '扫描方式',
    ADD COLUMN IF NOT EXISTS vuln_url VARCHAR(500) COMMENT '漏洞地址',

    -- 处置信息
    ADD COLUMN IF NOT EXISTS priority VARCHAR(20) COMMENT '处置优先级',
    ADD COLUMN IF NOT EXISTS false_positive BOOLEAN DEFAULT FALSE COMMENT '是否误报';

-- 更新 protocol 字段为实际端口的协议
UPDATE security_vulnerabilities sv
LEFT JOIN security_assets sa ON sv.asset_id = sa.id
SET sv.protocol = COALESCE(sa.protocol, 'TCP')
WHERE sv.protocol IS NULL OR sv.protocol = '';

-- 根据严重程度更新处置优先级
UPDATE security_vulnerabilities
SET priority = CASE
    WHEN severity IN ('critical', 'high') THEN '高'
    WHEN severity = 'medium' THEN '中'
    ELSE '低'
END
WHERE priority IS NULL;
