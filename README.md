# Rastreia

Plataforma de rastreio de entregas. A transportadora cadastra as entregas, o motorista atualiza o status pelo celular e o cliente acompanha tudo por um link público, em tempo real.

> Projeto em construção. Etapas prontas: base da API, segurança da base e eventos de status com rastreio público. Próxima: front-end.

## Stack

- **API:** Go, chi, pgx + sqlc, golang-migrate, Viper, JWT
- **Banco:** PostgreSQL
- **Testes:** testing + testify; integração com Postgres real via testcontainers
- **Infra:** Docker e Docker Compose

Próximas etapas: eventos de status e rastreio público, front-end em React + TypeScript, CI no GitHub Actions, tempo real com WebSocket e Redis, fila de notificações com RabbitMQ e deploy na AWS.

## Documentação

| Documento | Conteúdo |
| --- | --- |
| [PRD](docs/PRD.md) | O que o produto é, requisitos e o que está fora de escopo |
| [Arquitetura](docs/ARCHITECTURE.md) | Camadas, pastas, fluxo de requisição e visão alvo |
| [Regras](docs/RULES.md) | Regras de código, testes, git e definição de pronto |
| [Design](docs/DESIGN.md) | Convenções da API e das interfaces |
| [Segurança](docs/SECURITY.md) | Como cada risco é tratado e o que falta |
| [Tarefas](docs/TASKS.md) | Backlog por etapa |
| [Memória](docs/MEMORY.md) | Decisões tomadas e armadilhas conhecidas |

## Rodando local

Precisa só de Docker.

```bash
docker compose up --build
```

A API sobe em `http://localhost:8080` e já cria um admin de teste:

| E-mail | Senha |
| --- | --- |
| admin@rastreia.dev | admin12345 |

Se a porta 5432 já estiver ocupada por um Postgres instalado no seu PC, suba o banco em outra porta com `DB_PORT=5433 docker compose up --build`.

A API roda com [air](https://github.com/air-verse/air) dentro do Docker, com o código montado no container: a cada arquivo `.go`, `.sql` ou `.yaml` salvo, ela é recompilada e reiniciada sozinha. Essa configuração de desenvolvimento fica em `docker-compose.override.yml`, que o Docker Compose carrega automaticamente.

### Rodando o air direto na máquina

Suba só o banco e chame o air na raiz do projeto:

```bash
cp .env.example .env
docker compose up -d db
air
```

## Testes

```bash
make test        # unitários + integração (sobe um Postgres temporário no Docker)
make test-short  # só os unitários, sem Docker
```

## Exemplo de uso

```bash
# login
TOKEN=$(curl -s localhost:8080/auth/login \
  -d '{"email":"admin@rastreia.dev","password":"admin12345"}' | jq -r .token)

# cadastra um motorista
curl -s localhost:8080/drivers -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"João","email":"joao@rastreia.dev","password":"motorista1"}'

# cria uma entrega para ele (a chave evita duplicar a entrega se a requisição for repetida)
curl -s localhost:8080/deliveries -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: pedido-123" \
  -d '{"recipient_name":"Maria","recipient_email":"maria@example.com","address":"Rua A, 10","driver_id":2}'

# o motorista entra e vê as entregas dele
DRIVER=$(curl -s localhost:8080/auth/login \
  -d '{"email":"joao@rastreia.dev","password":"motorista1"}' | jq -r .token)
curl -s localhost:8080/me/deliveries -H "Authorization: Bearer $DRIVER"

# e atualiza o status
curl -s localhost:8080/deliveries/1/events -H "Authorization: Bearer $DRIVER" -d '{"status":"picked_up"}'

# o cliente acompanha pelo código de rastreio, sem login
curl -s localhost:8080/public/tracking/RS7K2M9QXA4P
```

A especificação OpenAPI completa fica em [`api/openapi.yaml`](api/openapi.yaml) e também é servida em `GET /openapi.yaml`.

## Endpoints

| Método | Rota | Quem usa |
| --- | --- | --- |
| GET | `/health` | público |
| POST | `/auth/login` | admin e motorista |
| GET, POST | `/drivers` | admin |
| GET, POST | `/deliveries` | admin |
| GET, PATCH | `/deliveries/{id}` | admin |
| GET, POST | `/deliveries/{id}/events` | admin e motorista dono da entrega |
| GET | `/me/deliveries` | motorista |
| GET | `/public/tracking/{code}` | público (30 req/min por IP) |

## Estrutura

```
cmd/api/              ponto de entrada da API
api/                  especificação OpenAPI
internal/
  auth/               JWT, senhas, login e middleware de papéis
  config/             configuração por variáveis de ambiente (Viper)
  database/           conexão, migrations e queries SQL
  store/              código gerado pelo sqlc
  delivery/           entregas, eventos de status, rastreio público e idempotência
  user/               motoristas e admin inicial
  httpx/, apperr/     helpers de HTTP e erros
  testdb/             Postgres temporário para os testes de integração
```

## Decisões

- **sqlc em vez de ORM:** as queries ficam em SQL puro e o Go é gerado com tipos, então erro de SQL aparece na hora de gerar e não em produção.
- **Status não muda pelo PATCH:** cada mudança de status é um evento com histórico (`delivery_events`), que alimenta o rastreio público e, depois, as notificações.
- **Código de rastreio sem 0/O e 1/I:** fica fácil de ditar por telefone.
