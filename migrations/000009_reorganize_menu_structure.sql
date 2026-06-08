-- 收敛一级菜单，并把 Jenkins / Consul 归入“变更发布”

UPDATE menus SET title = '工作台', updated_at = NOW()
WHERE menus.key = 'dashboard';

UPDATE menus SET title = '资产中心', icon = 'DatabaseOutlined', updated_at = NOW()
WHERE menus.key = 'cmdb';

UPDATE menus SET title = '变更发布', updated_at = NOW()
WHERE menus.key = 'deploy';

UPDATE menus SET title = '告警事件', updated_at = NOW()
WHERE menus.key = 'alarm';

UPDATE menus SET title = '事件中心', updated_at = NOW()
WHERE menus.key = 'alarm-events';

UPDATE menus SET title = '安全中心', updated_at = NOW()
WHERE menus.key = 'security';

UPDATE menus SET title = '应用管理', parent_id = (SELECT id FROM (SELECT id FROM menus WHERE menus.key = 'cmdb' LIMIT 1) AS t), sort = 4, updated_at = NOW()
WHERE menus.key = 'cmdb-applications';

UPDATE menus SET title = '迭代部署', updated_at = NOW()
WHERE menus.key = 'deploy-release';

UPDATE menus SET title = '部署记录', updated_at = NOW()
WHERE menus.key = 'deploy-history';

UPDATE menus SET title = '归档打包', updated_at = NOW()
WHERE menus.key = 'deploy-archive';

UPDATE menus SET title = '聚合打包', updated_at = NOW()
WHERE menus.key = 'deploy-aggregate-package';

UPDATE menus SET title = '安全资产', updated_at = NOW()
WHERE menus.key = 'security-assets';

UPDATE menus SET title = 'Jenkins任务', parent_id = (SELECT id FROM (SELECT id FROM menus WHERE menus.key = 'deploy' LIMIT 1) AS t), sort = 70, updated_at = NOW()
WHERE menus.key = 'jenkins';

UPDATE menus SET title = 'Consul配置变更', parent_id = (SELECT id FROM (SELECT id FROM menus WHERE menus.key = 'deploy' LIMIT 1) AS t), sort = 60, updated_at = NOW()
WHERE menus.key = 'consul';

UPDATE menus SET title = '批量配置下发', updated_at = NOW()
WHERE menus.key = 'consul-batch-all';

UPDATE menus SET title = '配置操作记录', updated_at = NOW()
WHERE menus.key = 'consul-operations';

UPDATE menus SET title = '配置管理', updated_at = NOW()
WHERE menus.key = 'consul-config';

UPDATE menus SET title = '视图管理', updated_at = NOW()
WHERE menus.key = 'jenkins-views';

UPDATE menus SET title = '我的资料', status = 0, updated_at = NOW()
WHERE menus.key = 'profile';
