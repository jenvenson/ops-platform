-- 历史扫描任务风险汇总回填
-- 2026-04-03
-- 口径：
-- 1. 不把资产识别结果计入 high_risk / medium_risk / low_risk
-- 2. 不把低置信度 host-version-match 候选结果计入任务风险汇总
-- 3. 仅回填漏洞扫描任务；资产发现任务保持原样

UPDATE `security_scan_tasks` AS t
LEFT JOIN (
  SELECT
    v.`task_id`,
    SUM(
      CASE
        WHEN COALESCE(v.`finding_family`, 'vulnerability') <> 'inventory'
          AND NOT (
            v.`finding_source` = 'host-version-match'
            AND COALESCE(v.`confidence`, '') <> 'high'
          )
          AND LOWER(COALESCE(v.`severity`, '')) IN ('critical', 'high')
        THEN 1 ELSE 0
      END
    ) AS `derived_high_risk`,
    SUM(
      CASE
        WHEN COALESCE(v.`finding_family`, 'vulnerability') <> 'inventory'
          AND NOT (
            v.`finding_source` = 'host-version-match'
            AND COALESCE(v.`confidence`, '') <> 'high'
          )
          AND LOWER(COALESCE(v.`severity`, '')) = 'medium'
        THEN 1 ELSE 0
      END
    ) AS `derived_medium_risk`,
    SUM(
      CASE
        WHEN COALESCE(v.`finding_family`, 'vulnerability') <> 'inventory'
          AND NOT (
            v.`finding_source` = 'host-version-match'
            AND COALESCE(v.`confidence`, '') <> 'high'
          )
          AND LOWER(COALESCE(v.`severity`, '')) IN ('low', 'info')
        THEN 1 ELSE 0
      END
    ) AS `derived_low_risk`
  FROM `security_vulnerabilities` AS v
  GROUP BY v.`task_id`
) AS stats
  ON stats.`task_id` = t.`id`
SET
  t.`high_risk` = COALESCE(stats.`derived_high_risk`, 0),
  t.`medium_risk` = COALESCE(stats.`derived_medium_risk`, 0),
  t.`low_risk` = COALESCE(stats.`derived_low_risk`, 0)
WHERE t.`scan_type` IN ('web', 'host-vuln', 'all');
