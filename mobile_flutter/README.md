# Mobile Flutter (base)

Base do app NeuroLife AI em Flutter.

## Estrutura inicial sugerida
- `lib/main.dart`
- `lib/app.dart`
- `lib/features/`
- `lib/core/`

## Proximo passo
Criar projeto Flutter real:
```bash
flutter create .
```

Depois, implementar:
1. Onboarding
2. Login
3. Home do dia
4. Captura rapida de tarefas

## Sprint 4 (iniciado)
- Tela de agenda conectada ao Gateway (`/api/v1/events`).
- Fluxos implementados em `lib/main.dart`:
	- Carregar eventos (GET)
	- Criar evento (POST)
	- Editar evento (PUT)
	- Excluir evento (DELETE)
	- Filtro por periodo (`from` e `to`)
	- Paginacao (`page` e `limit`)
- Para testar:
	1. Gere um JWT via endpoint de auth no gateway.
	2. Informe `Base URL` e `JWT Access Token` na tela.
	3. Use `Carregar Eventos` e `Criar Evento`.
