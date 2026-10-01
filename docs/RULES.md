# Regras do projeto

Regras para manter o projeto no eixo. Vale para todo código novo.

## Escopo

- Só entra o que está no [PRD](PRD.md). Ideia nova vai para "Depois" no [TASKS.md](TASKS.md).
- Uma etapa por vez. A etapa só termina com testes passando, README atualizado e código na `main`.
- Todo o código é original e escrito para este projeto.

## Go

- Formatação com `gofmt`; `go vet` sem avisos.
- Camadas: handler → service → store. Handler não fala com o banco; service não conhece HTTP.
- SQL só em `internal/database/queries`, gerado com `sqlc generate`. Nunca concatenar SQL.
- Mudança de banco só por migration nova. Migration publicada não se edita.
- Erros: serviço devolve erros de `apperr`; handler usa `httpx.WriteError`. Erro inesperado vira 500 sem detalhes para o cliente e com log no servidor.
- Toda entrada é validada no service, mesmo que o front já valide.
- Respostas nunca expõem campos internos (por exemplo `password_hash`). Usar structs de resposta.
- Configuração só por variável de ambiente, lida em `internal/config`.

## Front-end

- TypeScript estrito, sem `any`.
- Validação com o mesmo formato de erro da API (`fields`).
- Chamadas à API só pelo cliente HTTP central, com TanStack Query.

## Testes

- Toda regra de negócio nova tem teste unitário com testify.
- Rotas novas têm teste de handler; a partir da etapa 2, teste de integração com Postgres real (testcontainers).
- Bug corrigido ganha um teste que falhava antes da correção.

## Git

- Commits pequenos, em português, no imperativo ou descrevendo o que mudou.
- `main` sempre funcionando. Trabalho maior vai em branch e entra por PR com CI verde.
- `.env` nunca vai para o git. Segredos só por variável de ambiente.

## API

- Rotas e respostas documentadas em `api/openapi.yaml` no mesmo commit.
- JSON em `snake_case`. Datas em RFC 3339 (UTC).
- Códigos: 400 corpo malformado, 401 sem login, 403 sem permissão, 404 não encontrado ou não é seu, 409 conflito, 422 validação, 429 limite de requisições.

## Segurança

- Seguir o [SECURITY.md](SECURITY.md). Item novo de risco entra lá antes de entrar no código.
