SELECT id, event, result, target_name, duration_ms, created_at FROM go_operation_log WHERE event = 'deploy' ORDER BY id DESC LIMIT 3;
