# Rastreia

Plataforma de rastreio de entregas. A transportadora cadastra as entregas, o motorista atualiza o status pelo celular e o cliente acompanha tudo por um link público, em tempo real.

> Projeto em construção. Esta é a etapa 1: a base da API em Go.

## Stack

- **API:** Go, chi, pgx + sqlc, golang-migrate, Viper, JWT
- **Banco:** PostgreSQL
- **Testes:** testing + testify
- **Infra:** Docker e Docker Compose

Próximas etapas: eventos de status e rastreio público, front-end em React + TypeScript, CI no GitHub Actions, tempo real com WebSocket e Redis, fila de notificações com RabbitMQ e deploy na AWS.

## Rodando local

Precisa só de Docker.

```bash
docker compose up --build
```

A API sobe em `http://localhost:8080` e já cria um admin de teste:

| E-mail | Senha |
| --- | --- |
| admin@rastreia.dev | admin12345 |

Para rodar a API fora do Docker:

```bash
cp .env.example .env
docker compose up -d db
make run
```

## Testes

```bash
make test
```

## Exemplo de uso

```bash
# login
TOKEN=$(curl -s localhost:8080/auth/login \
  -d '{"email":"admin@rastreia.dev","password":"admin12345"}' | jq -r .token)

# cadastra um motorista
curl -s localhost:8080/drivers -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"João","email":"joao@rastreia.dev","password":"motorista1"}'

# cria uma entrega para ele
curl -s localhost:8080/deliveries -H "Authorization: Bearer $TOKEN" \
  -d '{"recipient_name":"Maria","recipient_email":"maria@example.com","address":"Rua A, 10","driver_id":2}'
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

## Estrutura

```
cmd/api/              ponto de entrada da API
api/                  especificação OpenAPI
internal/
  auth/               JWT, senhas, login e middleware de papéis
  config/             configuração por variáveis de ambiente (Viper)
  database/           conexão, migrations e queries SQL
  store/              código gerado pelo sqlc
  delivery/           regras e handlers de entregas
  user/               motoristas e admin inicial
  httpx/, apperr/     helpers de HTTP e erros
```

## Decisões

- **sqlc em vez de ORM:** as queries ficam em SQL puro e o Go é gerado com tipos, então erro de SQL aparece na hora de gerar e não em produção.
- **Status não muda pelo PATCH:** cada mudança de status vai virar um evento com histórico, que alimenta o rastreio público e as notificações.
- **Código de rastreio sem 0/O e 1/I:** fica fácil de ditar por telefone.
