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
- `GET /api/v1/gamification/summary`: resumo de XP, nivel e streak.
- `GET /api/v1/gamification/achievements`: lista de conquistas desbloqueadas.

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
