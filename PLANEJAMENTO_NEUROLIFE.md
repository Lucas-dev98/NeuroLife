# NeuroLife AI - Planejamento Completo de Produto, Arquitetura e Sprints

## 1. Visao do produto

NeuroLife AI sera um assistente pessoal inteligente, pensado para pessoas com TDAH, autismo e sobrecarga cognitiva. O objetivo e reduzir dependencia de memoria, automatizar organizacao do dia a dia e evitar esquecimentos em tarefas, contas, pausas e relacionamentos.

### Problemas que o produto resolve
- Esquecer compromissos e tarefas importantes.
- Acumular tarefas grandes e paralisar a execucao.
- Esquecer contas e perder controle financeiro.
- Nao manter rotina de pausas e autocuidado.
- Esquecer de enviar mensagens para pessoas proximas.
- Dificuldade de transformar intencao em acao.

### Proposta de valor
- Sistema unico com agenda, tarefas, financeiro, voz, automacao e IA.
- Lembretes multi-canal (app, push, e-mail, WhatsApp, calendario externo).
- Checklist inteligente que quebra tarefas grandes em etapas simples.
- IA contextual que sugere prioridades com base em tempo, energia e dinheiro.
- Fluxo de uso com baixa carga cognitiva.

## 2. Principios de design para TDAH e autismo

- Interface limpa, com foco em 1 acao principal por tela.
- Menos decisao manual e mais automacao assistida.
- Avisos progressivos e repeticao inteligente.
- Linguagem direta e sem excesso de informacao.
- Fracionamento automatico de tarefas grandes.
- Rotinas de pausa, hidratacao e descanso como parte do sistema.
- Feedback positivo continuo via gamificacao.
- Personalizacao de intensidade de lembretes para cada perfil.

## 3. Arquitetura definitiva (hibrida)

## 3.1 Tecnologias principais
- Frontend mobile: Flutter.
- Backend core: Go com Fiber.
- Servicos de IA: Python com FastAPI.
- Banco principal: PostgreSQL.
- Busca semantica/memoria: pgvector (no PostgreSQL).
- Cache e filas leves: Redis.
- Mensageria: RabbitMQ.
- Comunicacao interna: gRPC.
- Storage de arquivos (audio, comprovantes): MinIO (S3 compat).
- Infra: Docker no inicio, Kubernetes na escala.

## 3.2 Distribuicao por responsabilidade
- Go: autenticacao, agenda, tarefas, financeiro, notificacoes, gamificacao, relacionamentos.
- Python: planejamento inteligente, quebra de tarefas, voz, OCR, recomendacoes, previsoes.
- Flutter: experiencia de uso, captura rapida (voz/texto), dashboard e configuracoes.

## 3.3 Fluxo tecnico resumido
1. Usuario cria tarefa por texto ou voz no Flutter.
2. API Gateway (Go) autentica e roteia.
3. Servico de Agenda/Tarefas grava no PostgreSQL.
4. Evento vai para RabbitMQ para lembretes.
5. IA (Python) pode quebrar tarefa em subtarefas e ajustar prioridade.
6. Notificacoes (Go) dispara push/e-mail/WhatsApp nos horarios.
7. Modulo de gamificacao registra XP e conquistas.

## 4. Modulos funcionais e implementacao

## 4.1 Usuarios e autenticacao
### O que faz
- Cadastro, login, recuperacao de senha, login social, perfil e preferencias.

### Como sera feito
- Go + Fiber para endpoints de auth.
- JWT para sessao de acesso.
- Refresh token com rotacao.
- PostgreSQL para usuarios e preferencias.
- Redis para revogacao e sessao rapida.

### Tabelas principais
- users
- user_profiles
- user_preferences
- auth_sessions
- devices

## 4.2 Agenda inteligente gamificada
### O que faz
- Calendario diario/semanal/mensal.
- Compromissos pontuais e recorrentes.
- Linha do tempo do dia.
- Pontuacao por conclusao e antecipacao.

### Como sera feito
- Servico Agenda (Go).
- Regra de recorrencia com RRULE.
- Sincronizacao opcional com Google/Outlook.
- Integracao com gamificacao ao concluir evento.

### Regras de XP iniciais
- Concluir tarefa: +50
- Concluir antes do prazo: +100
- Rotina diaria concluida: +25
- Conta paga em dia: +75
- Pausa feita no horario: +20

## 4.3 Sistema anti-esquecimento multi-canal
### O que faz
- Varios lembretes por tarefa/compromisso.
- Escalonamento por importancia.
- Entrega em varios canais.

### Como sera feito
- Servico Notificacoes (Go) com scheduler.
- RabbitMQ para fila de eventos de aviso.
- Canais:
  - Push (Firebase)
  - E-mail (provedor SMTP/API)
  - WhatsApp (Meta Cloud API ou Twilio)
  - Telegram (bot)
  - Google Calendar e Outlook
- Politica de lembrete por prioridade:
  - Baixa: 1h antes
  - Media: 1 dia, 6h, 1h
  - Alta: 2 dias, 1 dia, 6h, 1h, 15min

## 4.4 Checklist inteligente (quebra em etapas)
### O que faz
- Transforma tarefas grandes em passos pequenos e executaveis.

### Como sera feito
- Servico IA (Python + FastAPI).
- Prompt estruturado para decomposicao em subtarefas.
- Ajuste por contexto do usuario (tempo disponivel, energia, prazo).
- Go grava subtarefas e dependencias no banco.

### Exemplo
Entrada: Preparar apresentacao da faculdade
Saida:
- Definir tema
- Levantar referencias
- Escrever roteiro
- Criar slides
- Ensaiar
- Revisar

## 4.5 Planejador inteligente diario
### O que faz
- Sugere plano do dia com prioridades e blocos de foco.

### Como sera feito
- IA em Python analisa:
  - prazos
  - importancia
  - tempo estimado
  - historico de atrasos
  - janela de energia
  - contas a vencer
- Retorna agenda recomendada com ordem e pausas.

## 4.6 Modulo financeiro (contas e gastos)
### O que faz
- Contas a pagar com vencimento e status.
- Registro de gastos e receitas.
- Orcamento mensal por categoria.

### Como sera feito
- Servico Financeiro (Go).
- PostgreSQL para transacoes e contas.
- Alertas de vencimento pelo modulo de notificacoes.

### Tabelas principais
- accounts
- transactions
- categories
- bills
- recurring_bills
- budgets

## 4.7 IA financeira
### O que faz
- Sugestoes e alertas de saude financeira.

### Como sera feito
- Python com regras + modelos leves.
- Deteccao de:
  - aumento anormal por categoria
  - risco de saldo negativo
  - assinaturas recorrentes esquecidas
- Gera mensagens acionaveis, sem jargao tecnico.

## 4.8 Voz para capturar tarefas e agendar
### O que faz
- Usuario fala, sistema entende e agenda.

### Como sera feito
- Flutter grava audio curto.
- Python usa Whisper para transcricao.
- NLU transforma texto em intencao estruturada.
- Go grava tarefa/conta/lembrete no modulo correto.

### Exemplo de transformacao
Fala: lembrar de pagar internet dia 10
Estrutura:
- tipo: financeiro
- titulo: pagar internet
- data: 10/mes

## 4.9 Mensagens automaticas via WhatsApp
### O que faz
- Lembretes para contato com pessoas importantes.
- Mensagens agendadas recorrentes.

### Como sera feito
- Modulo Relacionamentos (Go) + Notificacoes (Go).
- Integracao com WhatsApp Business API.
- Regras configuraveis:
  - lembrar se ficar X dias sem contato
  - enviar mensagem em data/frequencia especifica

### Observacao importante
- Envio automatico depende da politica do provedor e consentimento do usuario.

## 4.10 Rotina de pausas e descanso
### O que faz
- Lembretes de pausa, agua, alongamento e retorno ao foco.

### Como sera feito
- Go controla timers de foco e pausa.
- Flutter mostra instrucoes simples de pausa.
- Gamificacao recompensa autocuidado.

### Regra inicial sugerida
- 50 min foco + 10 min pausa.
- Alerta de hidratacao a cada 2h.

## 4.11 Segundo cerebro (memoria inteligente)
### O que faz
- Armazena ideias, notas, audios, anexos e permite busca semantica.

### Como sera feito
- Python gera embeddings.
- pgvector para indexacao semantica.
- MinIO para anexos/audio.
- Busca por significado, nao apenas palavra exata.

## 5. Contratos entre servicos (alto nivel)

## 5.1 Go -> Python (gRPC)
- Decompor tarefa.
- Gerar plano diario.
- Analisar risco financeiro.
- Interpretar voz.
- Extrair dados de comprovante (OCR).

## 5.2 Eventos em RabbitMQ
- task.created
- task.due_soon
- bill.due_soon
- reminder.triggered
- task.completed
- break.missed
- relationship.inactive

## 6. Aplicativo Flutter (experiencia)

## 6.1 Telas MVP
- Onboarding e perfil neurodivergente (preferencias de aviso).
- Home do dia (3 prioridades, proximos avisos, pausa atual).
- Calendario e timeline.
- Tarefas com checklist.
- Financeiro (contas, gastos, resumo).
- Captura por voz.
- Configuracao de lembretes multi-canal.

## 6.2 Regras de UX
- Sempre mostrar proxima acao concreta.
- Limitar opcoes por etapa.
- Botao de captura rapida em destaque.
- Cores e iconografia consistentes para reduzir confusao.

## 7. Seguranca, privacidade e conformidade

- LGPD desde o MVP.
- Criptografia em transito (TLS) e em repouso para dados sensiveis.
- Consentimento explicito para WhatsApp e mensagens automatizadas.
- Controle de exportacao e exclusao de dados.
- Auditoria de acoes criticas (financeiro e mensagens).

## 8. Observabilidade e operacao

- Logs estruturados por servico.
- Tracing distribuido entre Go e Python.
- Metricas: taxa de lembrete entregue, taxa de tarefa concluida, contas pagas em dia.
- Alertas de falhas de fila, latencia e erro de integracao externa.

## 9. Estrategia de entrega por sprints

### Premissas
- Sprint de 2 semanas.
- Equipe base recomendada:
  - 1 Flutter
  - 1 Go
  - 1 Python IA
  - 1 QA/Produto (pode ser compartilhado no inicio)

### Definicao de pronto (DoD)
- Feature com testes unitarios e integracao basica.
- Logs e tratamento de erro implementados.
- Telemetria minima adicionada.
- Documentacao de endpoint/evento atualizada.

## 10. Planejamento detalhado de sprints (20 sprints)

## Sprint 1 - Fundacao tecnica
### Entregas
- Repositorios e padrao de arquitetura.
- CI/CD inicial.
- Docker Compose com PostgreSQL, Redis, RabbitMQ, MinIO.
- API Gateway em Go com healthcheck.
- App Flutter base com navegacao inicial.

### Tecnologia
- Go/Fiber, Flutter, Docker, PostgreSQL.

### Criterio de aceite
- Ambiente sobe com um comando e app conecta no gateway.

## Sprint 2 - Autenticacao e perfil
### Entregas
- Cadastro, login, refresh token.
- Perfil e preferencias de notificacao.
- Tela de onboarding simplificada.

### Tecnologia
- Go + PostgreSQL + Redis + Flutter.

### Criterio de aceite
- Usuario cria conta, faz login e salva preferencias.

## Sprint 3 - Agenda basica
### Entregas
- CRUD de eventos.
- Visualizacao diaria/semanal.
- Eventos recorrentes simples.

### Tecnologia
- Go + PostgreSQL + Flutter.

### Criterio de aceite
- Usuario cria evento recorrente e visualiza no calendario.

## Sprint 4 - Tarefas e checklist manual
### Entregas
- CRUD de tarefas e subtarefas.
- Prioridade, prazo e etiquetas.
- Conclusao com registro de historico.

### Tecnologia
- Go + PostgreSQL + Flutter.

### Criterio de aceite
- Usuario conclui tarefa e subtarefas com consistencia.

## Sprint 5 - Notificacoes push + e-mail
### Entregas
- Scheduler de lembretes.
- Templates de notificacao.
- Push e e-mail funcionando.

### Tecnologia
- Go + RabbitMQ + Firebase + provedor e-mail.

### Criterio de aceite
- Lembretes chegam no horario configurado.

## Sprint 6 - Gamificacao v1
### Entregas
- Motor de XP, niveis e streaks.
- Regras de pontuacao para tarefa/evento/pausa.
- Tela de progresso no app.

### Tecnologia
- Go + PostgreSQL + Flutter.

### Criterio de aceite
- Concluir tarefa gera XP e atualiza nivel.

## Sprint 7 - IA de checklist
### Entregas
- Servico Python de decomposicao de tarefas.
- Endpoint gRPC Go <-> Python.
- Sugestao de subtarefas no app.

### Tecnologia
- Python FastAPI + gRPC + Go.

### Criterio de aceite
- Tarefa grande vira checklist coerente em segundos.

## Sprint 8 - Captura por voz
### Entregas
- Gravacao de audio no app.
- Transcricao com Whisper.
- Conversao para tarefa/compromisso/conta.

### Tecnologia
- Flutter + Python (Whisper) + Go.

### Criterio de aceite
- Usuario fala e item aparece no modulo correto.

## Sprint 9 - Financeiro basico
### Entregas
- Contas a pagar, receitas e gastos.
- Categorias e dashboard mensal.
- Alertas de vencimento.

### Tecnologia
- Go + PostgreSQL + Flutter + Notificacoes.

### Criterio de aceite
- Conta com vencimento gera lembretes e status pago.

## Sprint 10 - IA financeira v1
### Entregas
- Deteccao de variacao anormal de gasto.
- Previsao simples de fluxo mensal.
- Insights no feed principal.

### Tecnologia
- Python + PostgreSQL.

### Criterio de aceite
- Sistema mostra ao menos 3 insights acionaveis por mes.

## Sprint 11 - Planejador inteligente diario
### Entregas
- Algoritmo de priorizacao (prazo x impacto x energia).
- Sugestao de blocos de foco com pausas.
- Replanejamento rapido no app.

### Tecnologia
- Python + Go + Flutter.

### Criterio de aceite
- Plano do dia gerado em menos de 3 segundos.

## Sprint 12 - Pausas e bem-estar
### Entregas
- Timer de foco/pausa.
- Alertas de hidratacao e alongamento.
- Recompensa por pausas concluida.

### Tecnologia
- Go + Flutter + Gamificacao.

### Criterio de aceite
- Usuario recebe e confirma pausas sem interromper o fluxo.

## Sprint 13 - Relacionamentos v1
### Entregas
- Cadastro de pessoas importantes.
- Regra de frequencia minima de contato.
- Aviso de contato pendente.

### Tecnologia
- Go + PostgreSQL + Flutter.

### Criterio de aceite
- Sistema avisa quando ultrapassa limite sem contato.

## Sprint 14 - WhatsApp integrado
### Entregas
- Integracao com provedor WhatsApp.
- Mensagens agendadas e lembretes de contato.
- Logs de entrega e falha.

### Tecnologia
- Go + API WhatsApp (Meta/Twilio).

### Criterio de aceite
- Mensagem teste enviada com rastreabilidade completa.

## Sprint 15 - Segundo cerebro v1
### Entregas
- Cadastro de notas e audios.
- Embeddings e busca semantica.
- Tela de busca inteligente.

### Tecnologia
- Python + pgvector + MinIO + Flutter.

### Criterio de aceite
- Busca retorna conteudo relevante por significado.

## Sprint 16 - OCR financeiro
### Entregas
- Upload de comprovantes/boletos.
- OCR para extrair valor, vencimento e beneficiario.
- Preenchimento automatico de conta/gasto.

### Tecnologia
- Python OCR + Go Financeiro + Flutter.

### Criterio de aceite
- Extracao correta em percentual minimo acordado (ex.: 85%).

## Sprint 17 - Anti-procrastinacao
### Entregas
- Deteccao de tarefas adiadas repetidamente.
- Sugestao de microacao de 5 minutos.
- Escalonamento de lembrete para tarefas criticas.

### Tecnologia
- Python + Go Notificacoes + Flutter.

### Criterio de aceite
- Sistema reduz tarefas atrasadas no piloto interno.

## Sprint 18 - Mapa de energia e sobrecarga
### Entregas
- Identificacao de horarios de maior rendimento.
- Indicadores de sobrecarga e risco de burnout.
- Sugestoes de ajuste de agenda.

### Tecnologia
- Python analitico + Flutter.

### Criterio de aceite
- Usuario recebe recomendacoes personalizadas semanais.

## Sprint 19 - Qualidade, seguranca e performance
### Entregas
- Hardening de seguranca.
- Testes E2E principais.
- Otimizacao de consultas e filas.
- Monitoramento e alertas finais.

### Tecnologia
- Stack completa.

### Criterio de aceite
- Meta de estabilidade para beta atingida.

## Sprint 20 - Beta fechado
### Entregas
- Publicacao para grupo piloto.
- Coleta de feedback estruturado.
- Ajustes de usabilidade e regras de IA.

### Tecnologia
- Stack completa + analytics de produto.

### Criterio de aceite
- KPIs minimos de adesao e retencao definidos para fase publica.

## 11. MVP recomendado para validar mercado (ate sprint 10)

### Escopo MVP
- Agenda + tarefas + checklist IA.
- Notificacoes push/e-mail/WhatsApp (minimo viavel).
- Voz para capturar tarefas.
- Financeiro basico (contas e gastos).
- Gamificacao basica.
- Sugestoes inteligentes iniciais.

### Resultado esperado no MVP
- Reducao real de esquecimentos.
- Aumento de tarefas concluidas.
- Contas pagas em dia com maior consistencia.
- Evidencia de valor para neurodivergentes.

## 12. KPIs de sucesso do produto

- Taxa de tarefas concluidas por semana.
- Percentual de contas pagas no prazo.
- Taxa de abertura de lembretes por canal.
- Tempo medio entre criacao e conclusao de tarefa.
- Frequencia de pausas cumpridas.
- Retencao semanal e mensal.
- NPS especifico para usuarios com TDAH/autismo.

## 13. Riscos e mitigacoes

- Dependencia de API externa (WhatsApp): manter fallback por push/e-mail/SMS.
- Custo de IA: usar cache, limites e inferencia otimizada.
- Complexidade de microservicos cedo demais: comecar modular monolith e evoluir para servicos separados quando necessario.
- Sobrecarga de funcionalidades: manter foco no MVP ate validar retencao.

## 14. Ordem pratica de execucao recomendada

1. Sprint 1 a 4: Base + auth + agenda + tarefas.
2. Sprint 5 a 8: notificacao + gamificacao + checklist IA + voz.
3. Sprint 9 e 10: financeiro + IA financeira (fecha MVP forte).
4. Sprint 11 em diante: modulos avancados (segundo cerebro, OCR, anti-procrastinacao, energia, sobrecarga).

---

Se quiser, o proximo passo e eu converter este planejamento em backlog pronto para execucao no formato Jira (epicos, historias, tarefas tecnicas e criterios de aceite por ticket).