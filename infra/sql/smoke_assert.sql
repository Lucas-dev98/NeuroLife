-- Validacao dos ultimos smoke tests
-- Uso: docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_assert.sql

SELECT correlation_id, status, retry_count, max_retries, dispatched_at, last_error
FROM notification_schedule
WHERE correlation_id LIKE 'smoke-%'
ORDER BY id DESC
LIMIT 20;

SELECT correlation_id, last_error, failed_at
FROM notification_dead_letter
WHERE correlation_id LIKE 'smoke-%'
ORDER BY id DESC
LIMIT 20;
