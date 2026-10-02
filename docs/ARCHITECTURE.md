# Arquitetura

Como o Rastreia é organizado hoje e para onde ele vai. Decisões e o motivo de cada uma ficam em [MEMORY.md](MEMORY.md).

## Visão geral (hoje)

```
cliente HTTP ──► API Go (chi) ──► PostgreSQL
                    │
                    └─ migrations embutidas no binário, aplicadas ao subir
```

Um único binário Go (`cmd/api`) serve a API REST em JSON. O banco é PostgreSQL. Tudo sobe com `docker compose up`.

## Visão alvo

```
             ┌──────────── React + TypeScript (admin, motorista, rastreio público)
             │                     │ HTTPS                │ WebSocket
             ▼                     ▼                      ▼
        CDN/estáticos  ──►  API Go  ◄──── pub/sub ────► Redis
                              │   │
                              │   └──► RabbitMQ ──► worker de notificações ──► e-mail
                              ▼
                          PostgreSQL
```

As peças entram uma por vez, na ordem de [TASKS.md](TASKS.md). Nenhuma peça nova entra sem uma necessidade concreta no [PRD](PRD.md).

## Estrutura de pastas

```
cmd/api/              ponto de entrada: config, banco, rotas e shutdown gracioso
api/                  especificação OpenAPI (embutida e servida em /openapi.yaml)
internal/
  config/             lê variáveis de ambiente (Viper), com .env como fallback
  database/           pool pgx e migrations (golang-migrate)
    migrations/       SQL versionado (up/down)
    queries/          SQL de origem do sqlc
  store/              código GERADO pelo sqlc — não editar à mão
  auth/               JWT, bcrypt, login e middlewares de autenticação e papel
  user/               cadastro e listagem de usuários (admin e motorista)
  delivery/           regras de entregas
  apperr/             erros de domínio (validação, não encontrado, conflito)
  httpx/              helpers HTTP: JSON, decode seguro, mapeamento de erros
  server/             montagem das rotas e middlewares globais
web/                  front-end (Vite, React, TypeScript, Tailwind)
  src/lib/            chamadas à API, status e formatação
  src/components/     componentes visuais (DESIGN.md)
  src/pages/          uma tela por rota
docs/                 esta documentação
```

## Camadas

Cada domínio (`user`, `delivery`, ...) segue o mesmo desenho:

```
handler  ──►  service  ──►  Store (interface)  ──►  store (sqlc)  ──►  PostgreSQL
 HTTP         regras         o que o service        queries geradas
 decode       validação      precisa do banco
 status       normalização
```

- **Handler:** só traduz HTTP. Faz o decode com `httpx.Decode`, chama o service e responde com `httpx.JSON` ou `httpx.WriteError`. Não tem regra de negócio.
- **Service:** valida e normaliza a entrada (`apperr.Validator`), aplica as regras e devolve tipos próprios do domínio (ex.: `user.User`, que nunca carrega o hash da senha).
- **Store:** interface pequena declarada no próprio pacote do service, com só os métodos que ele usa. Nos testes, é trocada por um fake em memória.
- **Transações:** quando uma operação escreve em mais de uma tabela, o service usa `Store.InTx`, que recebe um `Store` ligado à transação (em `delivery`, implementado por `PGStore` com `pgx.BeginFunc`). Erro dentro da função desfaz tudo.
- **store (sqlc):** gerado a partir de `internal/database/queries/*.sql`. Todo acesso ao banco passa por aqui, sempre com parâmetros.

## Fluxo de uma requisição autenticada

1. Middlewares globais: request ID, IP real, log, recover, timeout de 15 s.
2. `Tokens.Authenticate` valida o `Authorization: Bearer <jwt>` e põe os claims (`UserID`, `Role`) no contexto.
3. `RequireRole` barra quem não tem o papel exigido (403).
4. Handler → service → store.
5. Erros conhecidos viram 404, 409 ou 422 com detalhes por campo. Qualquer outro vira 500 genérico e é logado sem vazar para o cliente.

## Autenticação

- Senhas com bcrypt (custo padrão).
- JWT HS256 assinado com `JWT_SECRET` (mínimo de 32 caracteres), validade `JWT_TTL` (24h hoje), com `sub` = id do usuário e `role`.
- Papéis: `admin` e `driver`. O cliente final não tem conta; ele usa o código de rastreio.
- Melhorias planejadas (tokens curtos, refresh, revogação) estão em [SECURITY.md](SECURITY.md).

## Banco de dados

- Migrations em `internal/database/migrations`, embutidas com `go:embed` e aplicadas automaticamente quando a API sobe.
- Uma migration nunca é editada depois de ir para a `main`: cria-se outra.
- Tabelas atuais: `users`, `deliveries`, `delivery_events` (histórico de status) e `idempotency_keys` (chaves do `POST /deliveries`, válidas por 24h).
- A mudança de status usa concorrência otimista: o `UPDATE` só altera a linha se o status ainda for o que o service leu (`WHERE status = from_status`); se outro evento chegou antes, responde 409.
- `deliveries.completed_at` guarda quando a entrega foi entregue ou falhou pela última vez; o rastreio público expira 30 dias depois.
- O status da entrega é validado também por `CHECK` no banco, não só na aplicação.

## Configuração

Variáveis de ambiente (ver `.env.example`): `DATABASE_URL`, `JWT_SECRET`, `JWT_TTL`, `PORT`, `ADMIN_NAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `DB_PORT`. Variáveis de ambiente têm prioridade sobre o `.env`.

## Ambiente de desenvolvimento

- `docker compose up` usa `docker-compose.yml` + `docker-compose.override.yml`: a API roda com air dentro do container e recompila a cada arquivo salvo.
- A imagem de produção é o estágio final do `Dockerfile` (distroless, usuário não-root, binário estático).
- Também dá para rodar só o banco no Docker e o `air` direto na máquina.

## Testes

- `go test -race ./...` (`make test`).
- Services testados com fakes do `Store`; handlers testados com `httptest`.
- Testes de integração com Postgres real (testcontainers): `internal/testdb` sobe um container por pacote de teste e cria um banco novo, já migrado, para cada teste. Cobrem transações, concorrência (eventos e idempotência em paralelo) e a API inteira com IDOR. São pulados com `-short` (`make test-short`) ou sem Docker.
