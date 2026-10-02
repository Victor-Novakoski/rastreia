# Rastreia

[![CI](https://github.com/Victor-Novakoski/rastreia/actions/workflows/ci.yml/badge.svg?branch=develop)](https://github.com/Victor-Novakoski/rastreia/actions/workflows/ci.yml)

Plataforma de rastreio de entregas. A transportadora cadastra as entregas, o motorista atualiza o status pelo celular e o cliente acompanha tudo por um link público, em tempo real.

> Projeto em construção. Etapas prontas: base da API, segurança, eventos de status com rastreio público, front-end, CI, tempo real, notificações várias transportadoras (cada uma com seus dados, cadastro aberto e página inicial) e rota do motorista (endereço pelo CEP, etiqueta com QR-code, bipar os pacotes e ordem sugerida das paradas).

## Stack

- **API:** Go, chi, pgx + sqlc, golang-migrate, Viper, JWT
- **Front:** React, TypeScript, Vite, Tailwind CSS, React Router; mapas com Leaflet e OpenStreetMap, CEP pelo ViaCEP
- **Banco:** PostgreSQL; Redis para tempo real, rate limit e bloqueio de login entre instâncias
- **Mensageria:** RabbitMQ com outbox transacional, retry e fila de falhas; worker separado para e-mail e Web Push (PWA)
- **Testes:** testing + testify; integração com Postgres, Redis e RabbitMQ reais via testcontainers; Vitest + Testing Library no front
- **Infra:** Docker e Docker Compose

Atualização em tempo real por WebSocket na página de rastreio e no painel.

Próxima etapa: deploy na AWS.

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

A API sobe em `http://localhost:8080` e o front em `http://localhost:5173`. A página inicial leva a cada área: cadastro e painel da transportadora (`/transportadora`), app do motorista (`/motorista`) e rastreio público (`/rastreio`). Qualquer pessoa pode cadastrar uma transportadora; a API também cria uma "Transportadora Demo" para testar:

| E-mail | Senha |
| --- | --- |
| admin@rastreia.dev | admin12345 |

Os e-mails de notificação (um a cada mudança de status) caem no Mailpit, em `http://localhost:8025`. O painel do RabbitMQ fica em `http://localhost:15672` (guest/guest).

Para testar o aviso no celular (Web Push), gere as chaves com `go run ./cmd/worker vapid`, cole as duas linhas no `.env` e suba de novo. Em `http://localhost:5173/rastreio/<código>` aparece o botão "Ativar avisos" (no Chrome do PC já funciona; no celular o navegador exige HTTPS, então só depois do deploy). No iPhone o push só funciona com o site adicionado à tela de início.

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

No front (`web/`), com Node 22:

```bash
npm ci
npm run dev        # Vite em http://localhost:5173 (a API precisa estar no ar)
npm test           # Vitest
npm run lint && npm run typecheck
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
| POST | `/auth/signup` | público: cadastro da transportadora, já logando |
| POST | `/auth/login` | transportadora e motorista |
| POST | `/auth/refresh`, `/auth/logout` | transportadora e motorista (cookie de sessão) |
| GET | `/me` | transportadora e motorista: usuário e transportadora |
| GET | `/summary` | transportadora: números do painel |
| GET, POST | `/drivers` | transportadora |
| GET, POST | `/deliveries` | transportadora |
| GET, PATCH | `/deliveries/{id}` | transportadora |
| GET, POST | `/deliveries/{id}/events` | transportadora e motorista dono da entrega |
| GET | `/me/deliveries` | motorista |
| GET | `/public/tracking/{code}` | público (30 req/min por IP) |
| GET | `/public/tracking/{code}/live` | público, WebSocket com cada mudança |
| GET | `/live/deliveries` | transportadora, WebSocket do painel (token na primeira mensagem) |

Cada transportadora só enxerga os próprios motoristas e entregas; o que é de outra responde 404.

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
  user/               cadastro da transportadora, motoristas e /me
  httpx/, apperr/     helpers de HTTP e erros
  testdb/             Postgres temporário para os testes de integração
web/                  front-end em React: página inicial, painel da transportadora, app do motorista e rastreio público
```

## Decisões

- **sqlc em vez de ORM:** as queries ficam em SQL puro e o Go é gerado com tipos, então erro de SQL aparece na hora de gerar e não em produção.
- **Status não muda pelo PATCH:** cada mudança de status é um evento com histórico (`delivery_events`), que alimenta o rastreio público e, depois, as notificações.
- **Código de rastreio sem 0/O e 1/I:** fica fácil de ditar por telefone.
