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
             ┌──────────── React + TypeScript (transportadora, motorista, rastreio público)
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
cmd/worker/           worker de notificações: lê a fila do RabbitMQ e manda os e-mails
api/                  especificação OpenAPI (embutida e servida em /openapi.yaml)
internal/
  config/             lê variáveis de ambiente (Viper), com .env como fallback
  database/           pool pgx e migrations (golang-migrate)
    migrations/       SQL versionado (up/down)
    queries/          SQL de origem do sqlc
  store/              código GERADO pelo sqlc — não editar à mão
  auth/               JWT, bcrypt, login e middlewares de autenticação e papel
  user/               cadastro da transportadora, motoristas e /me
  delivery/           regras de entregas e do endereço
  route/              rota do dia do motorista: bipar pacotes, agrupar paradas e ordenar
  notify/             notificações: relay do outbox para o RabbitMQ, worker, e-mail e Web Push
  push/               inscrição do navegador no Web Push pela página de rastreio
  retention/          apaga os dados do destinatário depois do prazo de retenção (LGPD)
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
- JWT HS256 assinado com `JWT_SECRET` (mínimo de 32 caracteres), validade `JWT_TTL` (15 min por padrão), com `sub` = id do usuário, `role` e `cid` = id da transportadora.
- Papéis: `carrier` (quem toca a transportadora) e `driver`. O cliente final não tem conta; ele usa o código de rastreio.

## Várias transportadoras (tenants)

- Cada transportadora é uma linha em `carriers`. Usuários e entregas têm `carrier_id`, e todo usuário pertence a exatamente uma transportadora.
- O `carrier_id` vem do token, nunca do corpo da requisição. As listagens filtram por ele na query SQL; buscas por id conferem a transportadora no service (`visible`) e respondem 404 quando não bate.
- Atribuir motorista confere que ele é da mesma transportadora.
- O WebSocket do painel usa um tópico por transportadora (`deliveries:<id>`), escolhido a partir do token.
- `POST /auth/signup` cria a transportadora e o responsável num único `INSERT` (CTE): e-mail repetido não deixa transportadora órfã.
- `ADMIN_EMAIL`/`ADMIN_PASSWORD` criam a "Transportadora Demo" com essa conta, para um banco novo já ter login.
- Melhorias planejadas (tokens curtos, refresh, revogação) estão em [SECURITY.md](SECURITY.md).

## Notificações

```
API: troca de status ─┬─► delivery_events (mesma transação, published_at NULL)
                      │
relay (goroutine da API, a cada 1s)
  └─► lê eventos não publicados (FOR UPDATE SKIP LOCKED)
      └─► exchange rastreia.events (topic, delivery.status_changed)
          ├─► fila notifications.email ──► worker ──► SMTP
          └─► fila notifications.push  ──► worker ──► serviço de push do navegador (Web Push)
                 │ falhou: rejeita
                 ▼
              notifications.email.retry (espera 30s e volta)
              notifications.email.dead (depois de 5 tentativas ou mensagem inválida)
```

- `delivery_events` funciona como outbox: o evento e a marca "falta publicar" são gravados juntos, então nenhuma troca de status fica sem notificação, mesmo com o RabbitMQ fora do ar. Sem `RABBITMQ_URL` o relay não roda e os eventos ficam no banco.
- O relay publica com confirmação do RabbitMQ e só então marca `published_at`. Eventos com mais de 1 hora são descartados sem envio. Entrega é **pelo menos uma vez**: uma queda entre publicar e gravar repete o e-mail.
- Cada canal tem sua fila (com `.retry` e `.dead`) ligada à mesma exchange; um canal não atrasa nem derruba o outro.
- A mensagem leva nome e e-mail do destinatário, então o e-mail não precisa do banco. O push precisa: lê as inscrições em `push_subscriptions` (por isso o worker recebe `DATABASE_URL` quando tem as chaves VAPID).
- Web Push: na página de rastreio o destinatário toca em "Ativar avisos", o navegador registra `web/public/sw.js`, cria a inscrição com a chave pública VAPID (`GET /public/push/key`) e a manda para `POST /public/tracking/{code}/push`. O worker criptografa e assina (VAPID) cada aviso; inscrições que o serviço responde 404/410 são apagadas, e todas são apagadas quando a entrega é entregue. O front é um PWA instalável (`manifest.webmanifest`); no iPhone o push só funciona com o site adicionado à tela de início.

## Banco de dados

- Migrations em `internal/database/migrations`, embutidas com `go:embed` e aplicadas automaticamente quando a API sobe.
- Uma migration nunca é editada depois de ir para a `main`: cria-se outra.
- Tabelas atuais: `users`, `deliveries`, `delivery_events` (histórico de status e outbox das notificações), `push_subscriptions` (navegadores que pediram aviso) e `idempotency_keys` (chaves do `POST /deliveries`, válidas por 24h).
- A mudança de status usa concorrência otimista: o `UPDATE` só altera a linha se o status ainda for o que o service leu (`WHERE status = from_status`); se outro evento chegou antes, responde 409.
- `deliveries.anonymized_at` marca quando os dados do destinatário foram apagados pela retenção.
- `deliveries.completed_at` guarda quando a entrega foi entregue ou falhou pela última vez; o rastreio público expira 30 dias depois.
- O status da entrega é validado também por `CHECK` no banco, não só na aplicação.

## Configuração

Variáveis de ambiente (ver `.env.example`): `DATABASE_URL`, `JWT_SECRET`, `JWT_TTL`, `PORT`, `ADMIN_NAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `DB_PORT`, `REDIS_URL`, `RABBITMQ_URL`, `VAPID_PUBLIC_KEY`, `RETENTION_DAYS`. O worker usa `RABBITMQ_URL`, `TRACKING_URL`, `SMTP_*` e, para o push, `DATABASE_URL` e `VAPID_*`. Variáveis de ambiente têm prioridade sobre o `.env`.

## Ambiente de desenvolvimento

- `docker compose up` usa `docker-compose.yml` + `docker-compose.override.yml`: a API roda com air dentro do container e recompila a cada arquivo salvo.
- O compose sobe também Redis, RabbitMQ (painel em `http://localhost:15672`, guest/guest), o worker e o Mailpit, que captura os e-mails em `http://localhost:8025`.
- A imagem de produção é o estágio final do `Dockerfile` (distroless, usuário não-root, binário estático).
- Também dá para rodar só o banco no Docker e o `air` direto na máquina.

## Testes

- `go test -race ./...` (`make test`).
- Services testados com fakes do `Store`; handlers testados com `httptest`.
- Testes de integração com Postgres real (testcontainers): `internal/testdb` sobe um container por pacote de teste e cria um banco novo, já migrado, para cada teste. Cobrem transações, concorrência (eventos e idempotência em paralelo) e a API inteira com IDOR. São pulados com `-short` (`make test-short`) ou sem Docker.
