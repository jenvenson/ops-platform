-- 检查是否存在部署菜单，如果不存在则创建
INSERT INTO menus (title, key, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '自动化部署', 'deploy', '/deploy', 'RocketOutlined', 0, 3, 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM menus WHERE key = 'deploy');

-- 获取部署菜单的ID
SET @deploy_menu_id = (SELECT id FROM menus WHERE key = 'deploy');

-- 插入聚合打包菜单项（作为部署菜单的子项）
INSERT INTO menus (title, key, path, icon, parent_id, sort, status, created_at, updated_at)
SELECT '安装包聚合', 'deploy-aggregate-package', '/deploy/aggregate-package', 'InboxOutlined', @deploy_menu_id, 4, 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM menus WHERE key = 'deploy-aggregate-package');