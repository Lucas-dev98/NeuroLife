# Backlog Jira - NeuroLife AI

## Como usar este backlog
- Este backlog ja vem organizado por Epicos, Historias e Criterios de Aceite.
- Cada historia contem prioridade, estimativa (story points), sprint sugerida e dependencias.
- O padrao de criterios de aceite segue formato Given/When/Then para facilitar QA e automacao.

## Convencoes
- Prioridade: Highest, High, Medium.
- Story Points: sequencia Fibonacci (1, 2, 3, 5, 8, 13).
- Sprints: 1 a 20 (2 semanas cada).
- Labels base: neurolife, mvp, tdah-friendly, autism-friendly.

---

## EPIC NLF-EP01 - Fundacao de Plataforma e DevOps
Objetivo: Estabelecer base tecnica para desenvolvimento seguro, repetivel e observavel.

### NLF-101 - Setup de repositorios, padrao de codigo e CI
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 1
- Labels: platform, devops
- Dependencias: nenhuma
- Criterios de aceite:
  - Given um novo clone de repositorio, When o pipeline e executado, Then build e testes padrao concluem com sucesso.
  - Given um pull request, When validacoes rodam, Then lint e testes bloqueiam merge com falhas.

### NLF-102 - Ambiente local com Docker Compose (PostgreSQL, Redis, RabbitMQ, MinIO)
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 1
- Labels: platform, infra
- Dependencias: NLF-101
- Criterios de aceite:
  - Given ambiente limpo, When executar docker compose up, Then todos os servicos sobem saudaveis.
  - Given servicos ativos, When API Gateway iniciar, Then conexoes com banco e filas sao estabelecidas.

### NLF-103 - Observabilidade basica (logs estruturados e health checks)
- Tipo: Story
- Prioridade: High
- Story Points: 3
- Sprint sugerida: 1
- Labels: observability
- Dependencias: NLF-102
- Criterios de aceite:
  - Given requisicoes na API, When processadas, Then logs incluem correlation_id e nivel de severidade.
  - Given endpoint de health, When consultado, Then retorna status por dependencia critica.

---

## EPIC NLF-EP02 - Identidade, Autenticacao e Preferencias
Objetivo: Garantir acesso seguro e personalizacao da experiencia.

### NLF-201 - Cadastro e login com JWT + refresh token
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 2
- Labels: auth, mvp
- Dependencias: NLF-102
- Criterios de aceite:
  - Given usuario valido, When realizar login, Then recebe access token e refresh token.
  - Given refresh token valido, When solicitar renovacao, Then novo access token e emitido.

### NLF-202 - Recuperacao de senha por e-mail
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 2
- Labels: auth
- Dependencias: NLF-201
- Criterios de aceite:
  - Given e-mail cadastrado, When solicitar recuperacao, Then token temporario e enviado.
  - Given token valido, When redefinir senha, Then login com nova senha funciona.

### NLF-203 - Perfil do usuario e preferencias neurodivergentes
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 2
- Labels: profile, tdah-friendly, autism-friendly, mvp
- Dependencias: NLF-201
- Criterios de aceite:
  - Given usuario autenticado, When salvar preferencias, Then sistema persiste intensidade de lembretes e canais.
  - Given preferencias salvas, When abrir app novamente, Then configuracao e carregada automaticamente.

---

## EPIC NLF-EP03 - Agenda e Calendario Inteligente
Objetivo: Centralizar compromissos com recorrencia e visao temporal clara.

### NLF-301 - CRUD de eventos e compromissos
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 3
- Labels: agenda, mvp
- Dependencias: NLF-201
- Criterios de aceite:
  - Given usuario autenticado, When criar evento com data e hora, Then evento aparece na timeline do dia.
  - Given evento existente, When editar ou excluir, Then alteracao reflete imediatamente.

### NLF-302 - Recorrencia de eventos (diario, semanal, mensal)
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 3
- Labels: agenda, recurrence
- Dependencias: NLF-301
- Criterios de aceite:
  - Given evento recorrente semanal, When calendario do mes e aberto, Then ocorrencias sao exibidas corretamente.

### NLF-303 - Visao diaria, semanal e mensal no app Flutter
- Tipo: Story
- Prioridade: High
- Story Points: 8
- Sprint sugerida: 3
- Labels: mobile, agenda, mvp
- Dependencias: NLF-301
- Criterios de aceite:
  - Given eventos cadastrados, When alternar visualizacao, Then datas e horarios permanecem consistentes.

---

## EPIC NLF-EP04 - Tarefas e Checklist
Objetivo: Converter demandas em execucao clara, rastreavel e em etapas pequenas.

### NLF-401 - CRUD de tarefas com prioridade, prazo e categoria
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 4
- Labels: tasks, mvp
- Dependencias: NLF-201
- Criterios de aceite:
  - Given tarefa criada, When listar tarefas do dia, Then tarefa aparece ordenada por prioridade e prazo.

### NLF-402 - Subtarefas e checklist manual
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 4
- Labels: tasks, checklist, mvp
- Dependencias: NLF-401
- Criterios de aceite:
  - Given tarefa com subtarefas, When concluir item, Then progresso percentual da tarefa principal e atualizado.

### NLF-403 - Dependencias entre tarefas
- Tipo: Story
- Prioridade: Medium
- Story Points: 3
- Sprint sugerida: 4
- Labels: tasks
- Dependencias: NLF-401
- Criterios de aceite:
  - Given tarefa dependente, When tarefa predecessora nao concluida, Then sistema sinaliza bloqueio de execucao.

---

## EPIC NLF-EP05 - Notificacoes e Sistema Anti-Esquecimento
Objetivo: Minimizar esquecimentos com lembretes multi-canal e escalonamento.

### NLF-501 - Scheduler de lembretes com regras por prioridade
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 5
- Labels: notifications, mvp
- Dependencias: NLF-301, NLF-401
- Criterios de aceite:
  - Given tarefa de alta prioridade, When criada, Then sistema agenda lembretes em 2d, 1d, 6h, 1h e 15min.

### NLF-502 - Push notifications (Firebase)
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 5
- Labels: notifications, mobile, mvp
- Dependencias: NLF-501
- Criterios de aceite:
  - Given lembrete agendado, When horario e atingido, Then push e entregue ao dispositivo ativo.

### NLF-503 - Notificacoes por e-mail
- Tipo: Story
- Prioridade: High
- Story Points: 3
- Sprint sugerida: 5
- Labels: notifications, mvp
- Dependencias: NLF-501
- Criterios de aceite:
  - Given canal e-mail habilitado, When lembrete dispara, Then e-mail e enviado com titulo e horario da tarefa.

### NLF-504 - Integracao WhatsApp para lembretes
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 14
- Labels: notifications, whatsapp, mvp
- Dependencias: NLF-501
- Criterios de aceite:
  - Given usuario com consentimento ativo, When lembrete critico dispara, Then mensagem e enviada via provedor WhatsApp.

---

## EPIC NLF-EP06 - Gamificacao
Objetivo: Aumentar adesao e consistencia de rotina com feedback positivo.

### NLF-601 - Motor de XP e niveis
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 6
- Labels: gamification, mvp
- Dependencias: NLF-401
- Criterios de aceite:
  - Given conclusao de tarefa, When evento e registrado, Then XP e somado e nivel recalculado.

### NLF-602 - Streaks diarios e semanais
- Tipo: Story
- Prioridade: High
- Story Points: 3
- Sprint sugerida: 6
- Labels: gamification
- Dependencias: NLF-601
- Criterios de aceite:
  - Given usuario ativo em dias consecutivos, When concluir rotina, Then streak incrementa corretamente.

### NLF-603 - Conquistas por marcos
- Tipo: Story
- Prioridade: Medium
- Story Points: 3
- Sprint sugerida: 6
- Labels: gamification
- Dependencias: NLF-601
- Criterios de aceite:
  - Given meta de 10 tarefas concluidas, When marco e atingido, Then conquista e desbloqueada no perfil.

---

## EPIC NLF-EP07 - Assistente de Voz e NLP
Objetivo: Permitir captura rapida de intencoes por audio.

### NLF-701 - Captura e upload de audio no Flutter
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 8
- Labels: voice, mobile, mvp
- Dependencias: NLF-201
- Criterios de aceite:
  - Given permissao de microfone, When usuario gravar audio, Then arquivo e enviado com sucesso ao backend.

### NLF-702 - Transcricao com Whisper
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 8
- Labels: voice, ai, mvp
- Dependencias: NLF-701
- Criterios de aceite:
  - Given audio legivel, When processado, Then transcricao textual e retornada com confianca minima definida.

### NLF-703 - Extracao de intencao (tarefa, evento, conta)
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 8
- Labels: voice, ai, nlp, mvp
- Dependencias: NLF-702
- Criterios de aceite:
  - Given frase lembrarme de pagar internet dia 10, When interpretada, Then sistema cria item financeiro com data de vencimento correta.

---

## EPIC NLF-EP08 - Financeiro
Objetivo: Garantir controle de contas, receitas e despesas com foco em lembranca de pagamentos.

### NLF-801 - Cadastro de contas a pagar e recorrencia
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 9
- Labels: finance, mvp
- Dependencias: NLF-201
- Criterios de aceite:
  - Given conta recorrente mensal, When novo ciclo inicia, Then proximo vencimento e gerado automaticamente.

### NLF-802 - Registro de gastos e receitas por categoria
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 9
- Labels: finance, mvp
- Dependencias: NLF-201
- Criterios de aceite:
  - Given transacao informada, When salva, Then saldo e total por categoria sao atualizados.

### NLF-803 - Dashboard financeiro mensal
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 9
- Labels: finance, mobile, mvp
- Dependencias: NLF-802
- Criterios de aceite:
  - Given dados do mes, When abrir dashboard, Then usuario visualiza gastos por categoria e contas pendentes.

### NLF-804 - Alertas de vencimento de contas
- Tipo: Story
- Prioridade: Highest
- Story Points: 3
- Sprint sugerida: 9
- Labels: finance, notifications, mvp
- Dependencias: NLF-501, NLF-801
- Criterios de aceite:
  - Given conta com vencimento proximo, When regra de lembrete aplica, Then usuario recebe aviso em canais habilitados.

---

## EPIC NLF-EP09 - IA de Planejamento e Recomendacoes
Objetivo: Oferecer sugestoes acionaveis conforme cenario de tarefas e financeiro.

### NLF-901 - Quebra inteligente de tarefas em subtarefas
- Tipo: Story
- Prioridade: Highest
- Story Points: 8
- Sprint sugerida: 7
- Labels: ai, checklist, mvp
- Dependencias: NLF-401
- Criterios de aceite:
  - Given tarefa ampla, When solicitar sugestao, Then sistema retorna ao menos 4 subtarefas objetivas e ordenadas.

### NLF-902 - Planejador diario com prioridade, prazo e energia
- Tipo: Story
- Prioridade: High
- Story Points: 8
- Sprint sugerida: 11
- Labels: ai, planning
- Dependencias: NLF-301, NLF-401, NLF-901
- Criterios de aceite:
  - Given agenda e tarefas do dia, When gerar plano, Then sistema sugere sequencia com blocos de foco e pausa.

### NLF-903 - Insights financeiros inteligentes
- Tipo: Story
- Prioridade: High
- Story Points: 8
- Sprint sugerida: 10
- Labels: ai, finance, mvp
- Dependencias: NLF-802
- Criterios de aceite:
  - Given historico de gastos, When processar analise, Then sistema alerta variacoes anormais e risco de saldo insuficiente.

---

## EPIC NLF-EP10 - Relacionamentos e Mensagens Automatizadas
Objetivo: Apoiar manutencao de vinculos afetivos com lembretes e automacao consciente.

### NLF-1001 - Cadastro de pessoas importantes e frequencia de contato
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 13
- Labels: relationships
- Dependencias: NLF-201
- Criterios de aceite:
  - Given contato cadastrado com frequencia de 7 dias, When periodo expira sem interacao, Then sistema marca contato como pendente.

### NLF-1002 - Lembrete para contato social pendente
- Tipo: Story
- Prioridade: High
- Story Points: 3
- Sprint sugerida: 13
- Labels: relationships, notifications
- Dependencias: NLF-1001, NLF-501
- Criterios de aceite:
  - Given contato pendente, When janela de alerta inicia, Then usuario recebe lembrete com sugestao de mensagem.

### NLF-1003 - Mensagens WhatsApp agendadas para contatos
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 14
- Labels: relationships, whatsapp
- Dependencias: NLF-504, NLF-1001
- Criterios de aceite:
  - Given mensagem agendada e consentimento valido, When horario chega, Then envio e registrado como entregue ou falha.

---

## EPIC NLF-EP11 - Bem-Estar, Pausas e Anti-Sobrecarga
Objetivo: Reduzir exaustao e melhorar consistencia com rotinas de descanso.

### NLF-1101 - Timer de foco e pausa (50/10 configuravel)
- Tipo: Story
- Prioridade: High
- Story Points: 5
- Sprint sugerida: 12
- Labels: wellbeing, mvp
- Dependencias: NLF-501
- Criterios de aceite:
  - Given timer iniciado, When ciclo de foco termina, Then sistema alerta pausa automaticamente.

### NLF-1102 - Alertas de hidratacao e alongamento
- Tipo: Story
- Prioridade: Medium
- Story Points: 3
- Sprint sugerida: 12
- Labels: wellbeing
- Dependencias: NLF-1101
- Criterios de aceite:
  - Given periodo prolongado de foco, When limite de hidratacao e atingido, Then lembrete e enviado.

### NLF-1103 - Deteccao de sobrecarga (sinal precoce)
- Tipo: Story
- Prioridade: Medium
- Story Points: 8
- Sprint sugerida: 18
- Labels: wellbeing, ai
- Dependencias: NLF-902
- Criterios de aceite:
  - Given volume excessivo de tarefas e baixa conclusao, When regra de sobrecarga e acionada, Then usuario recebe recomendacao de replanejamento.

---

## EPIC NLF-EP12 - Segundo Cerebro, Busca Semantica e OCR
Objetivo: Registrar memoria pessoal e automatizar entrada de dados financeiros por documentos.

### NLF-1201 - Notas e anexos com armazenamento em objeto
- Tipo: Story
- Prioridade: Medium
- Story Points: 5
- Sprint sugerida: 15
- Labels: second-brain
- Dependencias: NLF-102
- Criterios de aceite:
  - Given anotacao com arquivo, When salvar, Then conteudo textual e anexo ficam vinculados ao usuario.

### NLF-1202 - Embeddings e busca semantica com pgvector
- Tipo: Story
- Prioridade: Medium
- Story Points: 8
- Sprint sugerida: 15
- Labels: second-brain, ai
- Dependencias: NLF-1201
- Criterios de aceite:
  - Given consulta em linguagem natural, When buscar, Then resultados relevantes sao retornados por similaridade semantica.

### NLF-1203 - OCR de boletos e comprovantes
- Tipo: Story
- Prioridade: Medium
- Story Points: 8
- Sprint sugerida: 16
- Labels: ocr, finance
- Dependencias: NLF-801
- Criterios de aceite:
  - Given imagem legivel de boleto, When processada, Then valor e vencimento sao extraidos para pre-preenchimento do cadastro.

---

## EPIC NLF-EP13 - Integracoes de Calendario Externo
Objetivo: Levar lembretes para fora do app e ampliar confiabilidade.

### NLF-1301 - Integracao Google Calendar (2-way basica)
- Tipo: Story
- Prioridade: Medium
- Story Points: 8
- Sprint sugerida: 17
- Labels: integration, calendar
- Dependencias: NLF-301
- Criterios de aceite:
  - Given conta conectada, When evento e criado no NeuroLife, Then evento aparece no Google Calendar.

### NLF-1302 - Integracao Outlook Calendar (2-way basica)
- Tipo: Story
- Prioridade: Medium
- Story Points: 8
- Sprint sugerida: 17
- Labels: integration, calendar
- Dependencias: NLF-301
- Criterios de aceite:
  - Given conta conectada, When evento e atualizado no app, Then alteracao sincroniza no Outlook.

---

## EPIC NLF-EP14 - Qualidade, Seguranca e Beta
Objetivo: Preparar produto para grupo fechado com confiabilidade operacional.

### NLF-1401 - Hardening de seguranca (LGPD, trilhas, consentimento)
- Tipo: Story
- Prioridade: High
- Story Points: 8
- Sprint sugerida: 19
- Labels: security, compliance
- Dependencias: NLF-201, NLF-504
- Criterios de aceite:
  - Given usuario solicita exclusao de dados, When processo e iniciado, Then dados pessoais sao removidos conforme politica definida.

### NLF-1402 - Testes E2E de fluxos criticos
- Tipo: Story
- Prioridade: High
- Story Points: 8
- Sprint sugerida: 19
- Labels: qa
- Dependencias: NLF-401, NLF-501, NLF-801
- Criterios de aceite:
  - Given build candidata, When suite E2E executar, Then fluxos criar tarefa, lembrar tarefa e pagar conta passam sem regressao.

### NLF-1403 - Beta fechado e coleta estruturada de feedback
- Tipo: Story
- Prioridade: Highest
- Story Points: 5
- Sprint sugerida: 20
- Labels: beta
- Dependencias: NLF-1402
- Criterios de aceite:
  - Given usuarios piloto ativos, When ciclo de 2 semanas termina, Then feedback e consolidado em backlog priorizado de melhorias.

---

## Backlog MVP (recomendado ate Sprint 10)
Historias MVP para primeira validacao de mercado:
- NLF-201, NLF-203
- NLF-301, NLF-303
- NLF-401, NLF-402
- NLF-501, NLF-502, NLF-503
- NLF-601
- NLF-701, NLF-702, NLF-703
- NLF-801, NLF-802, NLF-803, NLF-804
- NLF-901, NLF-903
- NLF-504 (versao minima de WhatsApp)

## Dependencias criticas
- Voz depende de autenticacao + upload + servico IA.
- Financeiro com alertas depende de notificacoes.
- WhatsApp depende de consentimento e provedor aprovado.
- Planejador inteligente depende de agenda + tarefas + historico.

## KPIs por fase
- MVP: taxa de tarefas concluidas, contas pagas em dia, abertura de lembretes.
- Pos-MVP: retencao semanal, uso de voz, reducao de tarefas atrasadas, taxa de pausas cumpridas.

## Definition of Ready (DoR) para historias
- Objetivo da historia claro em 1 frase.
- Dependencias mapeadas.
- Criterios de aceite testaveis.
- Mock ou referencia de UX quando houver tela.

## Definition of Done (DoD) para historias
- Codigo revisado por pares.
- Testes unitarios minimos criados.
- Logs e tratamento de erro implementados.
- Telemetria minima adicionada.
- Criterios de aceite validados por QA/PO.
