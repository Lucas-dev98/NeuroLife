#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:18080}"
POSTGRES_USER="${POSTGRES_USER:-neurolife}"
POSTGRES_DB="${POSTGRES_DB:-neurolife}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 nao encontrado. Instale o comando para executar este smoke test."
    exit 1
  fi
}

require_cmd curl
require_cmd jq
require_cmd docker

echo "[1/9] Gerando identidade de teste..."
TS="$(date -u +%Y%m%d%H%M%S)"
EMAIL="smoke-task-reminders-${TS}@neurolife.local"
PASSWORD="Teste12345!"
NAME="Smoke Task Reminders ${TS}"

echo "[2/9] Registrando usuario..."
REGISTER_PAYLOAD=$(jq -n --arg name "$NAME" --arg email "$EMAIL" --arg password "$PASSWORD" '{name:$name,email:$email,password:$password}')
REGISTER_RESP=$(curl -sS -X POST "${BASE_URL}/api/v1/auth/register" -H "Content-Type: application/json" -d "${REGISTER_PAYLOAD}")
TOKEN=$(echo "$REGISTER_RESP" | jq -r '.tokens.access_token // empty')
USER_ID=$(echo "$REGISTER_RESP" | jq -r '.user.id // empty')

if [[ -z "$TOKEN" || -z "$USER_ID" ]]; then
  echo "Falha ao registrar usuario. Resposta:"
  echo "$REGISTER_RESP"
  exit 1
fi

echo "[3/9] Criando tarefa com prioridade alta e prazo futuro..."
DUE_AT=$(date -u -d "+3 days" +"%Y-%m-%dT%H:%M:%SZ")
TASK_PAYLOAD=$(jq -n --arg due "$DUE_AT" '{title:"Smoke reminder task",description:"Validar pipeline task reminders",category:"work",priority:"high",due_at:$due,checklist_titles:["passo 1"]}')
TASK_RESP=$(curl -sS -X POST "${BASE_URL}/api/v1/tasks" -H "Content-Type: application/json" -H "Authorization: Bearer ${TOKEN}" -d "${TASK_PAYLOAD}")
TASK_ID=$(echo "$TASK_RESP" | jq -r '.task.id // .id // empty')

if [[ -z "$TASK_ID" ]]; then
  echo "Falha ao criar tarefa. Resposta:"
  echo "$TASK_RESP"
  exit 1
fi

echo "[4/9] Validando outbox de task..."
OUTBOX_COUNT=$(docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT COUNT(*) FROM reminder_outbox WHERE task_id = ${TASK_ID};" | tr -d '[:space:]')
if [[ "${OUTBOX_COUNT:-0}" -lt 1 ]]; then
  echo "Nenhum item de outbox para task_id=${TASK_ID}."
  exit 1
fi

echo "[5/9] Aguardando schedule da tarefa ser populada..."
for attempt in $(seq 1 20); do
  ACTIVE_COUNT=$(docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT COUNT(*) FROM notification_schedule WHERE task_id = ${TASK_ID} AND status IN ('scheduled','queued') AND canceled_at IS NULL;" | tr -d '[:space:]')
  if [[ "${ACTIVE_COUNT:-0}" -ge 5 ]]; then
    break
  fi
  sleep 1
done

if [[ "${ACTIVE_COUNT:-0}" -lt 5 ]]; then
  echo "Schedule nao populado conforme esperado (esperado >=5 para prioridade high, atual=${ACTIVE_COUNT:-0})."
  exit 1
fi

echo "[6/9] Atualizando tarefa para prioridade low (deve reduzir lembretes ativos)..."
UPDATE_PAYLOAD=$(jq -n --arg due "$DUE_AT" '{title:"Smoke reminder task",description:"Validar pipeline task reminders",category:"work",priority:"low",due_at:$due,checklist_titles:["passo 1"]}')
UPDATE_RESP=$(curl -sS -X PUT "${BASE_URL}/api/v1/tasks/${TASK_ID}" -H "Content-Type: application/json" -H "Authorization: Bearer ${TOKEN}" -d "${UPDATE_PAYLOAD}")
UPDATED_ID=$(echo "$UPDATE_RESP" | jq -r '.task.id // .id // empty')
if [[ -z "$UPDATED_ID" ]]; then
  echo "Falha ao atualizar tarefa. Resposta:"
  echo "$UPDATE_RESP"
  exit 1
fi

echo "[7/9] Aguardando reconciliacao de schedules..."
for attempt in $(seq 1 20); do
  ACTIVE_COUNT=$(docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT COUNT(*) FROM notification_schedule WHERE task_id = ${TASK_ID} AND status IN ('scheduled','queued') AND canceled_at IS NULL;" | tr -d '[:space:]')
  if [[ "${ACTIVE_COUNT:-0}" -eq 1 ]]; then
    break
  fi
  sleep 1
done

if [[ "${ACTIVE_COUNT:-0}" -ne 1 ]]; then
  echo "Quantidade de schedules ativos apos update deveria ser 1 para prioridade low. Atual=${ACTIVE_COUNT:-0}."
  exit 1
fi

echo "[8/9] Excluindo tarefa..."
DELETE_STATUS=$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE "${BASE_URL}/api/v1/tasks/${TASK_ID}" -H "Authorization: Bearer ${TOKEN}")
if [[ "$DELETE_STATUS" != "204" ]]; then
  echo "Falha ao excluir tarefa. HTTP ${DELETE_STATUS}."
  exit 1
fi

for attempt in $(seq 1 20); do
  ACTIVE_COUNT=$(docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT COUNT(*) FROM notification_schedule WHERE task_id = ${TASK_ID} AND status IN ('scheduled','queued') AND canceled_at IS NULL;" | tr -d '[:space:]')
  if [[ "${ACTIVE_COUNT:-0}" -eq 0 ]]; then
    break
  fi
  sleep 1
done

if [[ "${ACTIVE_COUNT:-0}" -ne 0 ]]; then
  echo "Schedules ativos ainda presentes apos delete. Atual=${ACTIVE_COUNT:-0}."
  exit 1
fi

echo "[9/9] Smoke test de task reminders concluido com sucesso."