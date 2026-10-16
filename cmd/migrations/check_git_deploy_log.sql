SELECT id, target_name AS branch, duration_ms, result, SUBSTRING(jms_output, -500) AS tail_500 FROM go_operation_log WHERE event = 'deploy' AND jms_output LIKE '%composer%' ORDER BY id DESC LIMIT 3;
