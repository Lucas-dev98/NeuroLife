-- Smoke test de WhatsApp
-- Uso: docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_whatsapp.sql

WITH u AS (
  INSERT INTO users (email, password_hash)
  VALUES ('smoke-whatsapp@neurolife.local', 'x')
  ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
  RETURNING id
), e AS (
  INSERT INTO events (
    user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets
  )
  SELECT
    u.id,
    'Smoke WhatsApp Event',
    'Evento para validar whatsapp real',
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
  target_whatsapp,
  max_retries
)
SELECT
  'smoke-whatsapp-' || extract(epoch from now())::bigint || '-' || floor(random()*100000)::int,
  'scheduled',
  'event.reminder',
  'upsert',
  e.user_id,
  e.id,
  'Smoke WhatsApp Reminder',
  'Teste de notificacao whatsapp real',
  'UTC',
  NOW() + interval '20 minute',
  NOW() + interval '50 minute',
  NOW() - interval '1 minute',
  10,
  '["whatsapp"]'::jsonb,
  '5511999999999',
  3
FROM e;
