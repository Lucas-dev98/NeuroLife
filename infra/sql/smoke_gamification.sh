#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:18080}"
POSTGRES_USER="${POSTGRES_USER:-neurolife}"
POSTGRES_DB="${POSTGRES_DB:-neurolife}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl nao encontrado. Instale curl para executar o smoke test."
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq nao encontrado. Instale jq para executar o smoke test."
  exit 1
fi

echo "[1/8] Gerando identidade de teste..."
TS="$(date -u +%Y%m%d%H%M%S)"
EMAIL="smoke-gamification-${TS}@neurolife.local"
PASSWORD="Teste12345!"
NAME="Smoke Gamification ${TS}"

echo "[2/8] Registrando usuario..."
REGISTER_PAYLOAD=$(jq -n \
  --arg name "$NAME" \
  --arg email "$EMAIL" \
  --arg password "$PASSWORD" \
  '{name:$name,email:$email,password:$password}')

REGISTER_RESP=$(curl -sS -X POST "${BASE_URL}/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "${REGISTER_PAYLOAD}")

TOKEN=$(echo "$REGISTER_RESP" | jq -r '.tokens.access_token // empty')
USER_ID=$(echo "$REGISTER_RESP" | jq -r '.user.id // empty')

if [[ -z "$TOKEN" || -z "$USER_ID" ]]; then
  echo "Falha ao registrar usuario. Resposta:"
  echo "$REGISTER_RESP"
  exit 1
fi

echo "[3/8] Criando evento para conclusao..."
START_AT=$(date -u -d "+1 hour" +"%Y-%m-%dT%H:%M:%SZ")
END_AT=$(date -u -d "+2 hour" +"%Y-%m-%dT%H:%M:%SZ")

EVENT_PAYLOAD=$(jq -n \
  --arg title "Smoke Gamification Event ${TS}" \
  --arg description "Evento para validar XP e streak" \
  --arg start_at "$START_AT" \
  --arg end_at "$END_AT" \
  '{title:$title,description:$description,start_at:$start_at,end_at:$end_at,timezone:"UTC",is_all_day:false,recurrence:{freq:"weekly",interval:1},reminder_offsets_minutes:[60,15]}')

EVENT_RESP=$(curl -sS -X POST "${BASE_URL}/api/v1/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d "${EVENT_PAYLOAD}")

EVENT_ID=$(echo "$EVENT_RESP" | jq -r '.id // empty')
if [[ -z "$EVENT_ID" ]]; then
  echo "Falha ao criar evento. Resposta:"
  echo "$EVENT_RESP"
  exit 1
fi

echo "[4/8] Concluindo evento (deve pontuar)..."
COMPLETE_RESP_1=$(curl -sS -X POST "${BASE_URL}/api/v1/events/${EVENT_ID}/complete" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$COMPLETE_RESP_1" | jq '{awarded_xp, was_already_completed, summary, unlocked}'

echo "[5/8] Repetindo conclusao (deve ser idempotente)..."
COMPLETE_RESP_2=$(curl -sS -X POST "${BASE_URL}/api/v1/events/${EVENT_ID}/complete" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$COMPLETE_RESP_2" | jq '{awarded_xp, was_already_completed, summary, unlocked}'

echo "[6/8] Lendo summary e achievements via API..."
SUMMARY_RESP=$(curl -sS -X GET "${BASE_URL}/api/v1/gamification/summary" \
  -H "Authorization: Bearer ${TOKEN}")
ACH_RESP=$(curl -sS -X GET "${BASE_URL}/api/v1/gamification/achievements" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$SUMMARY_RESP" | jq .
echo "$ACH_RESP" | jq .

echo "[7/8] Validando em SQL (user_gamification + gamification_events + user_achievements)..."
docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
SELECT user_id, xp, level, current_streak, longest_streak, last_activity_day, updated_at
FROM user_gamification
WHERE user_id = ${USER_ID};
"

docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
SELECT user_id, source, source_id, xp_delta, created_at
FROM gamification_events
WHERE user_id = ${USER_ID}
ORDER BY id DESC
LIMIT 5;
"

docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
SELECT user_id, achievement_key, title, unlocked_at
FROM user_achievements
WHERE user_id = ${USER_ID}
ORDER BY id DESC;
"

echo "[8/8] Smoke test da gamificacao concluido com sucesso."
