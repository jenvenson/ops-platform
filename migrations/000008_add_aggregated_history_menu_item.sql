-- 更新菜单表，添加聚合历史菜单项
INSERT INTO menus (title, key, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '聚合历史', 'aggregated-history', '/deploy/aggregated-history', 'HistoryOutlined',
       id, 6, 1, NOW(), NOW()
FROM menus
WHERE key = 'deploy'
AND NOT EXISTS (
    SELECT 1 FROM menus WHERE key = 'aggregated-history'
);