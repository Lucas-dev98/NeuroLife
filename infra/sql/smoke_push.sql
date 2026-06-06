-- Smoke test de push
-- Uso: docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_push.sql

WITH u AS (
  INSERT INTO users (email, password_hash)
  VALUES ('smoke-dispatcher@neurolife.local', 'x')
  ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
  RETURNING id
), e AS (
  INSERT INTO events (
    user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets
  )
  SELECT
    u.id,
    'Smoke Push Event',
    'Evento para validar push real',
    NOW() + interval '20 minute',
    NOW() + interval '50 minute',
    'UTC',
    false,
    '{"type":"none"}'::jsonb,
    '[10]'::jsonb
  FROM u
  RETURNING id, user_id
)
INSERT INTO notification_schedule (
  correlation_id,
  status,
  topic,
  action,
  user_id,
  event_id,
  title,
  description,
  timezone,
  start_at,
  end_at,
  trigger_at,
  offset_minutes,
  target_channels,
  target_push,
  max_retries
)
SELECT
  'smoke-push-' || extract(epoch from now())::bigint || '-' || floor(random()*100000)::int,
  'scheduled',
  'event.reminder',
  'upsert',
  e.user_id,
  e.id,
  'Smoke Push Reminder',
  'Teste de notificacao push real',
  'UTC',
  NOW() + interval '20 minute',
  NOW() + interval '50 minute',
  NOW() - interval '1 minute',
  10,
  '["push"]'::jsonb,
  'replace_with_real_device_token',
  3
FROM e;
