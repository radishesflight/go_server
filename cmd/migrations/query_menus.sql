SELECT id, name, code, path, parent_id, sort, status FROM go_admin_menus WHERE name LIKE '%apk%' OR name LIKE '%前端%' OR name LIKE '%代码部署%' OR name LIKE '%集成部署%' ORDER BY parent_id, sort;
