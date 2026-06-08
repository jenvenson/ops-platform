USE ops_platform;
SET NAMES utf8mb4;

-- =========================================================
-- 线上菜单同步脚本
-- 用途：
-- 1. 补齐线上缺失菜单
-- 2. 修正 security-overview 路径
-- 3. 给 admin / ops 角色补齐菜单权限
-- =========================================================

-- 1. 安装包聚合
INSERT INTO menus (title, `key`, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '安装包聚合', 'deploy-aggregate-package', '/deploy/aggregate-package', 'InboxOutlined', m.id, 5, 1, NOW(), NOW()
FROM menus m
WHERE m.`key` = 'deploy'
  AND NOT EXISTS (
    SELECT 1 FROM menus WHERE `key` = 'deploy-aggregate-package'
  );

-- 2. 聚合历史
INSERT INTO menus (title, `key`, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '聚合历史', 'aggregated-history', '/deploy/aggregated-history', 'HistoryOutlined', m.id, 6, 1, NOW(), NOW()
FROM menus m
WHERE m.`key` = 'deploy'
  AND NOT EXISTS (
    SELECT 1 FROM menus WHERE `key` = 'aggregated-history'
  );

-- 3. 资产中心
INSERT INTO menus (title, `key`, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '资产中心', 'security-assets', '/security/assets', 'DatabaseOutlined', m.id, 3, 1, NOW(), NOW()
FROM menus m
WHERE m.`key` = 'security'
  AND NOT EXISTS (
    SELECT 1 FROM menus WHERE `key` = 'security-assets'
  );

-- 4. 漏洞工单
INSERT INTO menus (title, `key`, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '漏洞工单', 'security-tickets', '/security/tickets', 'FileTextOutlined', m.id, 5, 1, NOW(), NOW()
FROM menus m
WHERE m.`key` = 'security'
  AND NOT EXISTS (
    SELECT 1 FROM menus WHERE `key` = 'security-tickets'
  );

-- 5. 修正安全概览路径
UPDATE menus
SET path = '/security/overview', updated_at = NOW()
WHERE `key` = 'security-overview'
  AND path <> '/security/overview';

-- 5.1 修正归档菜单标题
UPDATE menus
SET title = '归档打包', updated_at = NOW()
WHERE `key` = 'deploy-archive'
  AND title <> '归档打包';

-- 6. 查询菜单 ID
SET @menu_deploy_aggregate_package = (SELECT id FROM menus WHERE `key` = 'deploy-aggregate-package' LIMIT 1);
SET @menu_aggregated_history      = (SELECT id FROM menus WHERE `key` = 'aggregated-history' LIMIT 1);
SET @menu_security_assets         = (SELECT id FROM menus WHERE `key` = 'security-assets' LIMIT 1);
SET @menu_security_tickets        = (SELECT id FROM menus WHERE `key` = 'security-tickets' LIMIT 1);

-- 7. 给 admin 角色补齐权限
INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_deploy_aggregate_package, NOW()
FROM roles r
WHERE r.code = 'admin' AND @menu_deploy_aggregate_package IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_deploy_aggregate_package
  );

INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_aggregated_history, NOW()
FROM roles r
WHERE r.code = 'admin' AND @menu_aggregated_history IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_aggregated_history
  );

INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_security_assets, NOW()
FROM roles r
WHERE r.code = 'admin' AND @menu_security_assets IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_security_assets
  );

INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_security_tickets, NOW()
FROM roles r
WHERE r.code = 'admin' AND @menu_security_tickets IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_security_tickets
  );

-- 8. 给 ops 角色补齐权限
INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_deploy_aggregate_package, NOW()
FROM roles r
WHERE r.code = 'ops' AND @menu_deploy_aggregate_package IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_deploy_aggregate_package
  );

INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_aggregated_history, NOW()
FROM roles r
WHERE r.code = 'ops' AND @menu_aggregated_history IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_aggregated_history
  );

INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_security_assets, NOW()
FROM roles r
WHERE r.code = 'ops' AND @menu_security_assets IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_security_assets
  );

INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT r.id, @menu_security_tickets, NOW()
FROM roles r
WHERE r.code = 'ops' AND @menu_security_tickets IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = r.id AND rm.menu_id = @menu_security_tickets
  );

-- 9. 执行后检查
SELECT id, `key`, title, path, parent_id, sort, status
FROM menus
WHERE `key` IN (
  'deploy-aggregate-package',
  'aggregated-history',
  'security-assets',
  'security-tickets',
  'security-overview'
)
ORDER BY parent_id, sort, id;

SELECT r.code, m.`key`, m.title
FROM role_menus rm
JOIN roles r ON r.id = rm.role_id
JOIN menus m ON m.id = rm.menu_id
WHERE m.`key` IN (
  'deploy-aggregate-package',
  'aggregated-history',
  'security-assets',
  'security-tickets'
)
ORDER BY r.code, m.sort, m.id;
