# Arquitetura

Como o Rastreia é organizado. As decisões e o motivo de cada uma ficam em [DECISIONS.md](DECISIONS.md).

## Visão geral

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

Na API, Redis e RabbitMQ são opcionais. Sem `REDIS_URL`, o tempo real, o rate limit e o bloqueio de login ficam em memória, o que serve para uma instância só. Sem `RABBITMQ_URL`, o relay não roda e os eventos ficam guardados no banco. Tudo sobe com `docker compose up`.

O front é uma aplicação estática (Vite) que chama a API direto pela `VITE_API_URL`.

## Deploy (etapa 7)

Front estático; API e worker em containers atrás de HTTPS; PostgreSQL sem acesso de fora; segredos fora de arquivo. O que falta está em [TASKS.md](TASKS.md) e nos itens 1, 16, 20 e 25 de [SECURITY.md](SECURITY.md). Nenhuma peça nova entra sem uma necessidade concreta no [PRD](PRD.md).

## Estrutura de pastas

```
cmd/api/              ponto de entrada: config, banco, rotas, tarefas em segundo plano e shutdown gracioso
cmd/worker/           worker de notificações: lê as filas do RabbitMQ e manda e-mail e Web Push (`worker vapid` gera as chaves)
api/                  especificação OpenAPI (embutida e servida em /openapi.yaml)
internal/
  config/             lê variáveis de ambiente (Viper), com .env como fallback
  database/           pool pgx e migrations (golang-migrate)
    migrations/       SQL versionado (up/down)
    queries/          SQL de origem do sqlc
  store/              código GERADO pelo sqlc — não editar à mão
  auth/               JWT, refresh token, bcrypt, login e middlewares de autenticação e papel
  user/               cadastro da transportadora, motoristas e /me
  delivery/           entregas, endereço, eventos de status, busca, rastreio público, idempotência, resumo do painel e as rotas WebSocket
  route/              rota do dia do motorista: bipar pacotes, agrupar paradas e ordenar
  realtime/           WebSocket e pub/sub do tempo real (em memória ou pelo Redis)
  notify/             notificações: relay do outbox para o RabbitMQ, worker, e-mail e Web Push
  push/               inscrição do navegador no Web Push pela página de rastreio
  retention/          anonimiza os dados do destinatário depois do prazo (LGPD) e apaga o que venceu
  apperr/             erros de domínio (validação, não encontrado, conflito)
  httpx/              helpers HTTP: JSON, decode seguro, mapeamento de erros e IP do cliente
  server/             montagem das rotas, middlewares globais e log de requisições
  testdb/, testredis/ Postgres e Redis temporários para os testes de integração
web/                  front-end (Vite, React, TypeScript, Tailwind)
  public/             service worker do push (sw.js), manifest do PWA e ícones
  src/lib/            chamadas à API, sessão, tempo real, status e formatação
  src/components/     componentes visuais (DESIGN.md)
  src/pages/          uma tela por rota
docs/                 esta documentação
```

## Camadas

Cada domínio (`user`, `delivery`, `route`, ...) segue o mesmo desenho:

```
handler  ──►  service  ──►  Store (interface)  ──►  store (sqlc)  ──►  PostgreSQL
 HTTP         regras         o que o service        queries geradas
 decode       validação      precisa do banco
 status       normalização
```

- **Handler:** só traduz HTTP. Faz o decode com `httpx.Decode`, chama o service e responde com `httpx.JSON` ou `httpx.WriteError`. Não tem regra de negócio.
- **Service:** valida e normaliza a entrada (`apperr.Validator`), aplica as regras e devolve tipos próprios do domínio (ex.: `user.User`, que nunca carrega o hash da senha).
- **Store:** interface pequena declarada no próprio pacote do service, com só os métodos que ele usa. Nos testes, é trocada por um fake em memória.
- **Transações:** quando uma operação escreve em mais de uma tabela, o service usa `Store.InTx`, que recebe um `Store` ligado à transação (em `delivery` e `route`, implementado por `PGStore` com `pgx.BeginFunc`). Erro dentro da função desfaz tudo.
- **store (sqlc):** gerado a partir de `internal/database/queries/*.sql`. Todo acesso ao banco passa por aqui, sempre com parâmetros.

## Fluxo de uma requisição autenticada

1. Middlewares globais, nesta ordem: request ID, IP do cliente pelo `X-Forwarded-For` (só com `TRUST_PROXY=true`), log da requisição, recover, headers de segurança, CORS e rate limit global (120 por minuto por IP). As rotas HTTP têm ainda um timeout de 15 s; os WebSockets ficam fora dele.
2. `Tokens.Authenticate` valida o `Authorization: Bearer <jwt>` e põe os claims (`UserID`, `Role`, `CarrierID`) no contexto e no log da requisição.
3. `RequireRole` barra quem não tem o papel exigido (403).
4. Handler → service → store. O service confere se o recurso é de quem pede.
5. Erros conhecidos viram 404, 409 ou 422 (com os campos). Rota que não existe responde 404 e método errado 405, no mesmo JSON. Cliente que desistiu recebe 499 e consulta que passou dos 15 s, 504. Qualquer outro erro vira um 500 genérico e é logado sem vazar para o cliente.

## Autenticação

- Senhas com bcrypt (custo padrão).
- JWT HS256 assinado com `JWT_SECRET` (mínimo de 32 caracteres), validade `JWT_TTL` (15 min por padrão), com `sub` = id do usuário, `role` e `cid` = id da transportadora.
- Sessão: o access token fica só em memória no front, e o refresh token, opaco e rotativo, vai no cookie `rastreia_refresh` (`HttpOnly`, `SameSite=Strict`, `Path=/auth`), válido por `REFRESH_TTL` (7 dias) sem uso. Detalhes em [SECURITY.md](SECURITY.md) #14 e #18.
- Papéis: `carrier` (quem toca a transportadora) e `driver`. Quem recebe a entrega não tem conta; usa o código de rastreio.

## Várias transportadoras (tenants)

- Cada transportadora é uma linha em `carriers`. Usuários e entregas têm `carrier_id`, e todo usuário pertence a exatamente uma transportadora.
- O `carrier_id` vem do token, nunca do corpo da requisição. As listagens filtram por ele na query SQL; buscas por id conferem a transportadora no service (`visible`) e respondem 404 quando não bate.
- O motorista só alcança as entregas atribuídas a ele: `/me/deliveries` filtra na query, e `GET /deliveries/{id}` e os eventos respondem 404 para as outras.
- Atribuir motorista confere que ele é da mesma transportadora.
- O WebSocket do painel usa um tópico por transportadora (`deliveries:<id>`), escolhido a partir do token.
- `POST /auth/signup` cria a transportadora e o responsável num único `INSERT` (CTE): e-mail repetido não deixa transportadora órfã.
- `ADMIN_EMAIL`/`ADMIN_PASSWORD` criam a "Transportadora Demo" com essa conta, para um banco novo já ter login.

## Rota do motorista

- A rota é do motorista e do dia (fuso de São Paulo). Bipar um pacote (`POST /me/route/deliveries`) aceita o código ou o link do QR-code; pacote sem motorista passa a ser dele, de outro motorista responde 409 e de outra transportadora 404.
- Pacotes no mesmo CEP, rua e número viram uma parada; endereços sem número só se juntam quando estão no mesmo ponto do mapa.
- A ordem sugerida (`POST /me/route/optimize`) é vizinho mais próximo seguido de 2-opt, em linha reta, a partir de onde o motorista está. A ordem é guardada por pacote em `route_items`, e as paradas são montadas na leitura.

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

- `delivery_events` funciona como outbox: o evento e a marca "falta publicar" são gravados juntos, então nenhuma troca de status fica sem notificação, mesmo com o RabbitMQ fora do ar por um tempo.
- O relay publica com confirmação do RabbitMQ e só então marca `published_at`. Aviso com mais de 1 hora não vale mais a pena: o relay e o worker descartam em vez de mandar atrasado. A entrega é **pelo menos uma vez**: uma queda entre publicar e gravar repete o e-mail.
- Cada canal tem a sua fila (com `.retry` e `.dead`) ligada à mesma exchange; um canal não atrasa nem derruba o outro. Sem as chaves VAPID, o worker esvazia a fila de push em vez de deixá-la crescer.
- A mensagem leva o nome e o e-mail do destinatário, então o e-mail não precisa do banco. O push precisa: lê as inscrições em `push_subscriptions` (por isso o worker recebe `DATABASE_URL` quando tem as chaves VAPID).
- Web Push: na página de rastreio, quem recebe toca em "Ativar avisos", o navegador registra `web/public/sw.js`, cria a inscrição com a chave pública VAPID (`GET /public/push/key`) e a manda para `POST /public/tracking/{code}/push`. O worker criptografa e assina (VAPID) cada aviso; inscrições que o serviço de push responde 404/410 são apagadas, e todas são apagadas quando a entrega é entregue. O front é um PWA instalável (`manifest.webmanifest`); no iPhone, o push só funciona com o site adicionado à tela de início.

## Front

- Aplicação estática em React, com React Router e TanStack Query (cache, invalidação depois de cada mudança e tentativas com sinal ruim).
- A sessão é conferida só nas telas que dependem dela: a página inicial, as de entrar e cadastrar, o painel e o app do motorista. Abrir o rastreio público não chama a API por sessão.
- O access token fica em memória e é renovado no 401; a renovação passa por uma trava entre abas (`navigator.locks`), porque usar o mesmo refresh token duas vezes derruba a sessão.
- O tempo real abre um WebSocket por tela e reconecta sozinho, esperando cada vez mais (até 30 s); depois de uma queda, a tela busca de novo o que pode ter perdido.
- Mapas (Leaflet) e o leitor de QR-code (jsQR) são baixados só nas telas que usam. Se não baixam, o resto da tela continua funcionando.
- O build injeta a Content-Security-Policy no `index.html` (`web/csp.ts`).

## Banco de dados

- Migrations em `internal/database/migrations`, embutidas com `go:embed` e aplicadas automaticamente quando a API sobe.
- Uma migration nunca é editada depois de ir para a `main`: cria-se outra.
- Tabelas: `carriers` (transportadoras, os tenants), `users` (responsável e motoristas, com `carrier_id`), `deliveries` (entregas, com o endereço em partes e as coordenadas), `delivery_events` (histórico de status e outbox das notificações), `idempotency_keys` (chaves do `POST /deliveries`, válidas por 24h), `refresh_tokens` (sessões, só o hash), `push_subscriptions` (navegadores que pediram aviso), `routes` e `route_items` (a rota do dia de cada motorista e a ordem dos pacotes).
- A mudança de status usa concorrência otimista: o `UPDATE` só altera a linha se o status ainda for o que o service leu (`WHERE status = from_status`); se outro evento chegou antes, responde 409.
- `deliveries.completed_at` guarda quando a entrega foi entregue ou falhou pela última vez; o rastreio público expira 30 dias depois.
- `deliveries.anonymized_at` marca quando os dados do destinatário foram apagados pela retenção. A partir daí, a entrega não aceita mais alteração.
- `CHECK` no banco garante o status e o papel válidos e que latitude e longitude vêm juntas, não só a aplicação.

## Configuração

Variáveis da API (ver `.env.example`): `DATABASE_URL`, `JWT_SECRET`, `JWT_TTL`, `REFRESH_TTL`, `PORT`, `APP_ENV`, `CORS_ORIGINS`, `TRUST_PROXY`, `ADMIN_NAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `REDIS_URL`, `RABBITMQ_URL`, `VAPID_PUBLIC_KEY` e `RETENTION_DAYS`. O worker usa `APP_ENV`, `RABBITMQ_URL`, `TRACKING_URL`, `SMTP_*` e, para o push, `DATABASE_URL` e `VAPID_*`. O front usa `VITE_API_URL` (`web/.env.example`). `DB_PORT`, `REDIS_PORT` e `RABBITMQ_PORT` só mudam as portas publicadas pelo docker compose. Variáveis de ambiente têm prioridade sobre o `.env`.

## Ambiente de desenvolvimento

- `docker compose up` usa `docker-compose.yml` + `docker-compose.override.yml`: API e worker rodam com air dentro do container e recompilam a cada arquivo salvo, e o front roda no Vite com hot reload em `http://localhost:5173`.
- O compose sobe também Postgres, Redis, RabbitMQ (painel em `http://localhost:15672`, guest/guest) e o Mailpit, que captura os e-mails em `http://localhost:8025`.
- A imagem de produção é o estágio final do `Dockerfile` (distroless, usuário não-root, binários estáticos da API e do worker).
- Também dá para rodar só as dependências no Docker e o `air` direto na máquina (README).

## Testes

- `go test -race ./...` (`make test`).
- Services testados com fakes do `Store`; handlers testados com `httptest`.
- Testes de integração com Postgres, Redis e RabbitMQ reais (testcontainers): `internal/testdb` sobe um Postgres por pacote de teste e cria um banco novo, já migrado, para cada teste; `internal/testredis` faz o mesmo com os bancos lógicos do Redis. Cobrem transações, concorrência (eventos e idempotência em paralelo), a rotação do refresh token, o tempo real e o rate limit divididos entre duas instâncias pelo Redis, as filas do worker, a rota do motorista e a API inteira com IDOR. São pulados com `-short` (`make test-short`) ou sem Docker.
- `TestOpenAPI` falha se uma rota servida não estiver em `api/openapi.yaml` (ou o contrário).
- Front: Vitest + Testing Library (`npm test` em `web/`).
- A CI roda tudo isso em todo PR (`.github/workflows/ci.yml`).
