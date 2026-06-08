# NeuroLife AI

Monorepo inicial da plataforma NeuroLife AI.

## Estrutura
- `services/gateway-go`: API Gateway em Go (Fiber).
- `services/ai-python`: Servico de IA em Python (FastAPI).
- `mobile_flutter`: App mobile Flutter.
- `infra`: artefatos de infraestrutura local (migracoes, observabilidade futura).

## Requisitos
- Docker e Docker Compose

## Subir ambiente local
```bash
docker compose up --build
```

Servicos disponiveis:
- Gateway Go: `http://localhost:18080`
- Health detalhado: `http://localhost:18080/healthz`
- AI Python: `http://localhost:18000`
- Postgres: interno no Docker network (sem porta publica por padrao)
- Redis: `localhost:6379`
- RabbitMQ: `localhost:5672` (painel em `http://localhost:15672`)
- MinIO: `http://localhost:9000` (console em `http://localhost:9001`)
- Outbox Worker: processa `reminder_outbox` e publica na fila `reminder.triggered`
- Notifier Consumer: consome `reminder.triggered` e agenda notificacoes reais em `notification_schedule`
- Notifier Dispatcher: le `notification_schedule` por `trigger_at`, despacha por canal e aplica retry/dead-letter

## Endpoints iniciais
### Gateway
- `GET /healthz`: status do gateway e dependencias.
- `GET /api/v1/ping`: endpoint de teste.
- `POST /api/v1/auth/register`: cadastro de usuario.
- `POST /api/v1/auth/login`: login de usuario.
- `POST /api/v1/auth/refresh`: rotacao de refresh token.
- `GET /api/v1/profile`: perfil do usuario autenticado.
- `PUT /api/v1/profile`: atualiza nome do perfil.
- `GET /api/v1/preferences`: preferencias de lembrete.
- `PUT /api/v1/preferences`: atualiza preferencias de lembrete.
- `POST /api/v1/events`: cria evento com recorrencia inicial.
- `GET /api/v1/events`: lista eventos do usuario (filtros `from` e `to` opcionais, RFC3339).
- `GET /api/v1/events/:id`: detalhe do evento.
- `PUT /api/v1/events/:id`: atualiza evento.
- `DELETE /api/v1/events/:id`: exclui evento (soft delete).
- `POST /api/v1/events/:id/complete`: conclui evento e aplica XP/streak.
- `POST /api/v1/tasks`: cria tarefa com checklist inicial opcional.
- `GET /api/v1/tasks`: lista tarefas do usuario (paginacao por `page` e `limit`).
- `GET /api/v1/tasks/:id`: detalhe da tarefa.
- `PUT /api/v1/tasks/:id`: atualiza tarefa.
- `DELETE /api/v1/tasks/:id`: exclui tarefa (soft delete).
- `POST /api/v1/tasks/:id/checklist`: adiciona item de checklist.
- `PATCH /api/v1/tasks/:id/checklist/:itemId`: atualiza status de item de checklist.
- `DELETE /api/v1/tasks/:id/checklist/:itemId`: remove item de checklist.
- `GET /api/v1/gamification/summary`: resumo de XP, nivel e streak.
- `GET /api/v1/gamification/achievements`: lista de conquistas desbloqueadas.

### Dependencias entre tarefas (Sprint 4 - NLF-403)
- Campo opcional em criacao/atualizacao de tarefa: `depends_on_task_id`.
- Regra: o `depends_on_task_id` deve apontar para outra tarefa valida do mesmo usuario.
- Resposta da API inclui:
	- `depends_on_task_id`
	- `is_blocked` (true quando a tarefa predecessora ainda nao foi concluida).
	- `next_reminders` (ate 3 proximos lembretes agendados para a tarefa).

### Lembretes por tarefa (Sprint 5 - NLF-501)
- As tarefas com `due_at` geram contratos de lembrete na `reminder_outbox`.
- O pipeline `worker -> notifier-consumer -> notifier-dispatcher` agora suporta `task_id` alem de `event_id`.
- Offsets padrao por prioridade:
	- `low`: 60 min
	- `medium`: 1440, 360, 60 min
	- `high`: 2880, 1440, 360, 60, 15 min
	- `urgent`: 2880, 1440, 360, 60, 15, 5 min
- Ao atualizar tarefa com prazo, os lembretes antigos sao cancelados e os novos sao reprocessados.
- Ao excluir tarefa (ou remover prazo), os lembretes pendentes da tarefa sao cancelados.

## Mensageria de lembretes
- Fila RabbitMQ: `reminder.triggered`
- Producer: `outbox-worker` (separado do gateway)
- Consumer: `notifier-consumer` (agendamento em banco)
- Dispatcher: `notifier-dispatcher` (entrega final + retry + dead-letter)
- Fonte dos dados: tabela `reminder_outbox`
- Schema JSON: `infra/reminder.triggered.schema.json`

### Retry e dead-letter
- Tentativas controladas por `retry_count` e `max_retries` em `notification_schedule`.
- Backoff exponencial: 1m, 2m, 4m, ...
- Falha definitiva move o item para `notification_dead_letter` e marca status `dead_letter`.

### Gamificacao (Sprint 6)
- Tabelas novas: `user_gamification`, `gamification_events`, `user_achievements`.
- Regras iniciais ao concluir evento (`POST /api/v1/events/:id/complete`):
	- +50 XP por conclusao.
	- +25 XP extra se concluido no prazo (`completed_at <= end_at`).
- Nivel calculado por faixa de XP: `nivel = (xp / 500) + 1`.
- Streak diario:
	- mesma data: mantem streak.
	- dia seguinte: incrementa streak.
	- lacuna maior que 1 dia: reinicia para 1.

### Smoke test da gamificacao (HTTP + SQL)
- Script unico: `infra/sql/smoke_gamification.sh`
- Executa fluxo completo:
	- cadastro de usuario
	- login
	- criacao de evento
	- conclusao do evento (incluindo idempotencia)
	- leitura de summary/achievements por API
	- validacao em SQL (`user_gamification`, `gamification_events`, `user_achievements`)
- Execucao:
```bash
./infra/sql/smoke_gamification.sh
```

### Smoke test de lembretes por tarefa (HTTP + SQL)
- Script unico: `infra/sql/smoke_task_reminders.sh`
- Executa fluxo completo:
	- cadastro de usuario
	- criacao de tarefa com prioridade `high`
	- validacao do outbox para `task_id`
	- validacao de schedules ativos
	- update para prioridade `low` (reconciliacao para 1 lembrete ativo)
	- exclusao da tarefa e cancelamento dos schedules ativos
- Execucao:
```bash
./infra/sql/smoke_task_reminders.sh
```

### Mobile (Flutter) com gamificacao
- A tela inicial de agenda agora inclui:
	- botao `Carregar Gamificacao`
	- card com XP, nivel, streak atual e maior streak
	- lista de conquistas desbloqueadas
	- acao `Concluir` em cada evento para chamar `POST /api/v1/events/:id/complete`

### Adaptadores reais de canais
- O dispatcher suporta `push` (FCM HTTP), `email` (SMTP) e `whatsapp` (provider HTTP).
- Modos de execucao (`DISPATCH_CHANNEL_MODE`):
	- `mixed` (padrao): usa provider real quando configurado e fallback para simulacao quando faltar credencial.
	- `real`: exige provider configurado; se faltar configuracao, a notificacao falha e entra no fluxo de retry/dead-letter.
	- `mock`: sempre simulado, sem chamadas externas.
- Variaveis principais:
	- Push: `FCM_ENDPOINT`, `FCM_AUTH_BEARER`, `FCM_DEFAULT_TOKEN`, `FCM_TIMEOUT_SECONDS`.
	- Email: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`.
	- WhatsApp: `WHATSAPP_API_URL`, `WHATSAPP_API_TOKEN`, `WHATSAPP_DEFAULT_TO`, `WHATSAPP_TIMEOUT_SECONDS`.
- Destinos por notificacao (opcional):
	- `notification_schedule.target_push`
	- `notification_schedule.target_email`
	- `notification_schedule.target_whatsapp`
	- Se vazios, o dispatcher usa fallbacks por ambiente (`FCM_DEFAULT_TOKEN`, email do usuario para SMTP, `WHATSAPP_DEFAULT_TO`).

### Smoke test dos canais reais
1. Crie o arquivo local com credenciais reais:
```bash
cp .env.dispatcher.real.example .env.dispatcher.real
```
2. Edite `.env.dispatcher.real` com credenciais reais dos providers.
3. Suba o dispatcher usando esse arquivo:
```bash
docker compose --env-file .env.dispatcher.real up -d notifier-dispatcher
```
4. Dispare smoke tests SQL por canal:
```bash
docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_push.sql
docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_email.sql
docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_whatsapp.sql
```
5. Verifique resultado final (dispatched/dead_letter):
```bash
docker compose exec -T postgres psql -U neurolife -d neurolife -f infra/sql/smoke_assert.sql
```

### AI Service
- `GET /healthz`: status do servico de IA.
- `POST /v1/decompose-task`: quebra tarefa em subtarefas simples (mock inicial).

## Proximo passo (Sprint 2)
- Auth (cadastro/login/refresh).
- Persistencia de usuarios no PostgreSQL.
- Perfil e preferencias neurodivergentes.
