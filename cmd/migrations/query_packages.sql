SHOW TABLES LIKE 'go_code%';
SELECT '--- go_code_packages (latest 5) ---' AS '';
SELECT id, name, version, full_name, size, file_url, status, created_at FROM go_code_packages ORDER BY id DESC LIMIT 5;
SELECT '--- go_code_package_deploy_targets (latest 5) ---' AS '';
SELECT id, package_id, asset_id, asset_path_id, sort, status, created_at FROM go_code_package_deploy_targets ORDER BY id DESC LIMIT 5;
