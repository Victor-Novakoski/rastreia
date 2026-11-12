# Arquitetura

Visão geral de como o Rastreia é organizado. O porquê de cada escolha está em [decisoes/](decisoes/); as regras e armadilhas de cada parte ficam no `CLAUDE.md` da pasta dela.

## Peças

```
navegador (React + TypeScript, PWA)
   │ HTTP/JSON              │ WebSocket
   ▼                        ▼
API Go (cmd/api) ◄── pub/sub, rate limit e bloqueio de login ──► Redis
   │        │
   │        └─ relay do outbox (a cada 1 s) ──► RabbitMQ ──► worker (cmd/worker) ──► e-mail (SMTP)
   │                                                            │       └────────► Web Push
   ▼                                                            │
PostgreSQL ◄────────────── inscrições de push ──────────────────┘
```

Dois binários Go saem da mesma imagem Docker. `cmd/api` serve a API REST em JSON e os WebSockets, aplica as migrations ao subir e roda duas tarefas em segundo plano: o relay do outbox e a retenção de dados. `cmd/worker` consome as filas do RabbitMQ e manda o e-mail e o Web Push.

Na API, Redis e RabbitMQ são opcionais. Sem `REDIS_URL`, o tempo real, o rate limit e o bloqueio de login ficam em memória, o que serve para uma instância só. Sem `RABBITMQ_URL`, o relay não roda e os eventos ficam guardados no banco.

O front é uma aplicação estática (Vite) que chama a API direto pela `VITE_API_URL`.

## Camadas

Cada domínio (`user`, `delivery`, `route`, ...) segue o mesmo desenho:

```
handler  ──►  service  ──►  Store (interface)  ──►  store (sqlc)  ──►  PostgreSQL
 HTTP         regras         o que o service        queries geradas
 decode       validação      precisa do banco
 status       normalização
```

O handler só traduz HTTP, o service aplica as regras e devolve tipos do domínio, e todo acesso ao banco passa pelo código gerado pelo sqlc, sempre com parâmetros.

## Várias transportadoras

Cada transportadora é um tenant. Usuários e entregas têm `carrier_id`, que vem sempre do token, nunca do corpo da requisição. O que é de outra transportadora responde 404. O motorista só alcança as entregas atribuídas a ele, e quem recebe a entrega não tem conta: acompanha pelo código de rastreio.

## Notificações

```
API: troca de status ─┬─► delivery_events (mesma transação, published_at NULL)
                      │
relay (goroutine da API, a cada 1 s)
  └─► lê eventos não publicados (FOR UPDATE SKIP LOCKED)
      └─► exchange rastreia.events (topic, delivery.status_changed)
          ├─► fila notifications.email ──► worker ──► SMTP
          └─► fila notifications.push  ──► worker ──► serviço de push do navegador (Web Push)

em cada fila, se o envio falha:
  <fila>.retry   espera 30 s e devolve para a fila
  <fila>.dead    depois de 5 tentativas, ou na hora para mensagem inválida
```

`delivery_events` é ao mesmo tempo o histórico do rastreio e o outbox: nenhuma troca de status fica sem notificação, mesmo com o RabbitMQ fora do ar por um tempo. A entrega é pelo menos uma vez.

## Pastas

```
cmd/api, cmd/worker   pontos de entrada
api/                  especificação OpenAPI (servida em /openapi.yaml)
internal/             back-end em Go, um pacote por domínio
web/                  front-end (Vite, React, TypeScript, Tailwind)
docs/                 esta documentação
```

## Deploy

Front estático; API e worker em containers atrás de HTTPS; PostgreSQL sem acesso de fora; segredos fora de arquivo. Sem custo de hospedagem ([decisão 097](decisoes/097-deploy-sem-custo-de-hospedagem.md)). O que falta está nas [issues](https://github.com/Victor-Novakoski/rastreia/issues).
