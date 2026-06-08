-- 历史漏洞结果回填：来源、家族、置信度、主 CVE、漏洞库关联
-- 2026-04-03
-- 仅回填空字段，可重复执行

UPDATE `security_vulnerabilities`
SET `primary_cve_id` = UPPER(TRIM(SUBSTRING_INDEX(`cve_id`, ',', 1)))
WHERE (`primary_cve_id` IS NULL OR `primary_cve_id` = '')
  AND `cve_id` IS NOT NULL
  AND TRIM(`cve_id`) <> '';

UPDATE `security_vulnerabilities` AS v
LEFT JOIN `security_assets` AS a ON a.`id` = v.`asset_id`
SET v.`finding_source` = CASE
  WHEN v.`scan_method` = '服务识别' OR v.`vuln_type` LIKE '%资产识别%' THEN 'asset-inventory'
  WHEN v.`scanner` = 'vuln-matcher' THEN 'host-version-match'
  WHEN v.`template_id` LIKE 'web-rule-%' THEN 'web-rule'
  WHEN v.`scanner` = 'nuclei' AND (
    LOWER(COALESCE(a.`service_name`, '')) LIKE '%http%'
    OR LOWER(COALESCE(a.`service_name`, '')) LIKE '%https%'
    OR LOWER(COALESCE(a.`service_name`, '')) LIKE '%ssl%'
    OR v.`port` IN (80, 443, 8000, 8008, 8080, 8081, 8082, 8083, 8084, 8085, 8088, 8090, 8443, 8888, 9000, 9001, 9002, 9090, 3000, 5000, 7000, 7001)
  ) THEN 'web-template'
  WHEN v.`scanner` = 'nuclei' THEN 'host-template'
  ELSE v.`finding_source`
END
WHERE v.`finding_source` IS NULL OR v.`finding_source` = '';

UPDATE `security_vulnerabilities`
SET `finding_family` = CASE
  WHEN `finding_source` = 'asset-inventory' THEN 'inventory'
  ELSE 'vulnerability'
END
WHERE `finding_family` IS NULL OR `finding_family` = '';

UPDATE `security_vulnerabilities`
SET `match_mode` = CASE
  WHEN `finding_source` = 'web-rule' THEN 'rule'
  WHEN `finding_source` = 'asset-inventory' THEN 'inventory'
  WHEN `finding_source` IN ('web-template', 'host-template') THEN 'template'
  WHEN `finding_source` = 'host-version-match' AND (`matched_on` LIKE 'CPE%' OR `matched_on` LIKE '% < %') THEN 'version-range'
  WHEN `finding_source` = 'host-version-match' THEN 'fuzzy-product'
  ELSE `match_mode`
END
WHERE `match_mode` IS NULL OR `match_mode` = '';

UPDATE `security_vulnerabilities`
SET `match_mode` = 'version-range'
WHERE `finding_source` = 'host-version-match'
  AND `match_mode` = 'fuzzy-product'
  AND (`matched_on` LIKE 'CPE%' OR `matched_on` LIKE '% < %');

UPDATE `security_vulnerabilities`
SET `confidence` = CASE
  WHEN `finding_source` = 'asset-inventory' THEN 'low'
  WHEN `finding_source` = 'web-rule' THEN 'medium'
  WHEN `finding_source` = 'host-version-match' AND `match_mode` IN ('version-range', 'exact') THEN 'high'
  WHEN `finding_source` = 'host-version-match' THEN 'medium'
  WHEN `finding_source` IN ('web-template', 'host-template') AND (`primary_cve_id` IS NOT NULL AND `primary_cve_id` <> '') THEN 'high'
  WHEN `finding_source` IN ('web-template', 'host-template') AND LOWER(COALESCE(`severity`, '')) IN ('critical', 'high') THEN 'high'
  WHEN LOWER(COALESCE(`severity`, '')) = 'medium' THEN 'medium'
  ELSE 'low'
END
WHERE `confidence` IS NULL OR `confidence` = '';

UPDATE `security_vulnerabilities`
SET `confidence` = 'high'
WHERE `finding_source` = 'host-version-match'
  AND `match_mode` IN ('version-range', 'exact')
  AND `confidence` <> 'high';

UPDATE `security_vulnerabilities` AS v
JOIN `vulnerability_database` AS d
  ON UPPER(d.`cve_id`) = UPPER(v.`primary_cve_id`)
SET v.`vuln_db_id` = COALESCE(v.`vuln_db_id`, d.`id`),
    v.`cnvd_id` = CASE WHEN v.`cnvd_id` IS NULL OR v.`cnvd_id` = '' THEN d.`cnvd_id` ELSE v.`cnvd_id` END,
    v.`cnnvd_id` = CASE WHEN v.`cnnvd_id` IS NULL OR v.`cnnvd_id` = '' THEN d.`cnnvd_id` ELSE v.`cnnvd_id` END,
    v.`cncve_id` = CASE WHEN v.`cncve_id` IS NULL OR v.`cncve_id` = '' THEN d.`cncve_id` ELSE v.`cncve_id` END
WHERE v.`primary_cve_id` IS NOT NULL
  AND v.`primary_cve_id` <> ''
  AND (
    v.`vuln_db_id` IS NULL
    OR v.`cnvd_id` IS NULL OR v.`cnvd_id` = ''
    OR v.`cnnvd_id` IS NULL OR v.`cnnvd_id` = ''
    OR v.`cncve_id` IS NULL OR v.`cncve_id` = ''
  );
