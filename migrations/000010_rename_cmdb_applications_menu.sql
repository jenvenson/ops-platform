UPDATE menus
SET title = '应用流水线管理', updated_at = NOW()
WHERE menus.key = 'cmdb-applications';
