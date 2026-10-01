# Arquitetura

## Visão geral

```
 Painel admin ─┐
 Tela motorista├─ HTTPS ─▶ API Go ──▶ PostgreSQL
 Rastreio      ┘   WS  ◀──┤  │
                          │  ├──▶ Redis (pub/sub dos eventos)
                          │  └──▶ RabbitMQ ──▶ Worker Go ──▶ E-mail
```

- **API Go:** única porta de entrada. Autentica, valida, grava no Postgres e publica eventos.
- **PostgreSQL:** fonte da verdade. Tudo que importa está aqui.
- **Redis:** avisa todas as instâncias da API sobre eventos novos, para o WebSocket funcionar mesmo com mais de uma instância.
- **RabbitMQ + worker:** envia e-mails fora da requisição. Se o worker cair, as mensagens ficam na fila e são processadas quando ele voltar.
- **Front-end:** React + TypeScript + Vite + Tailwind + TanStack Query, servido como site estático.

## Fluxo principal: mudança de status

1. Motorista envia `POST /deliveries/{id}/events`.
2. A API confere se a entrega é dele, valida a transição de status e, numa transação, grava o evento e atualiza o status da entrega.
3. Depois do commit, publica o evento no Redis e na fila.
4. Instâncias da API recebem pelo Redis e mandam para quem está com a página de rastreio aberta.
5. O worker consome a fila, envia o e-mail e grava em `notifications`.

Se a publicação falhar depois do commit, o evento já está salvo e aparece no histórico. Para não perder notificação, o plano é usar o padrão outbox na etapa 6.

## Back-end

```
cmd/api/            main: lê config, roda migrations, sobe o servidor
internal/
  config/           Viper, variáveis de ambiente
  database/         pool pgx, migrations embutidas, queries SQL
  store/            código gerado pelo sqlc (não editar)
  auth/             JWT, bcrypt, login, middleware de papéis
  user/             motoristas e admin inicial
  delivery/         regras e handlers de entregas
  apperr/           erros de domínio (validação, não encontrado, conflito)
  httpx/            JSON, decode seguro, mapeamento de erros para HTTP
  server/           rotas chi e middlewares
api/openapi.yaml    contrato da API
```

Camadas: **handler** (HTTP, decode, status code) → **service** (regras e validação) → **store** (SQL gerado). O service depende de uma interface pequena do store, então os testes usam fakes em memória.

## Modelo de dados

| Tabela | Campos principais |
| --- | --- |
| users | id, name, email (único), password_hash, role (admin ou driver), created_at |
| deliveries | id, tracking_code (único), recipient_name, recipient_email, address, status, driver_id, created_at, updated_at |
| delivery_events | id, delivery_id, status, note, created_by, created_at |
| notifications | id, event_id, channel, sent_at, error |

Status possíveis: `pending → picked_up → in_transit → delivered`, e `failed` a partir de qualquer status em aberto.

## Ambientes

| Ambiente | Como sobe |
| --- | --- |
| Desenvolvimento | `docker compose up`: Postgres + API com air (hot reload) |
| Produção | Imagem distroless do estágio final do Dockerfile, numa EC2 com Docker Compose; Postgres no RDS; front no S3 + CloudFront. Depois, migração para ECS. |

## Decisões

Registradas em [MEMORY.md](MEMORY.md).
