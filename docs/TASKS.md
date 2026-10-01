# Tarefas

Backlog em ordem. Só se trabalha na etapa atual; o que surgir no caminho entra aqui antes de ser feito ([RULES.md](RULES.md#1-escopo)). Os números `#N` apontam para os itens de [SECURITY.md](SECURITY.md).

**Etapa atual: 1.5 — Segurança da base**

## Etapa 1 — Base da API ✅

- [x] Estrutura do projeto (chi, pgx, sqlc, golang-migrate, Viper)
- [x] Login com JWT e papéis admin/motorista
- [x] Cadastro e listagem de motoristas
- [x] CRUD de entregas com código de rastreio
- [x] OpenAPI servido em `/openapi.yaml`
- [x] Docker Compose com Postgres e porta configurável (`DB_PORT`)
- [x] Hot reload com air (no Docker e direto na máquina)
- [x] Documentação em `docs/`

## Etapa 1.5 — Segurança da base

Prioridade alta:
- [ ] Só confiar em `X-Forwarded-For` vindo de proxies conhecidos (`TRUSTED_PROXIES`) (#23)
- [ ] Rate limit global por IP e específico no login, com 429 e `Retry-After` (#8, #15)
- [ ] Tempo constante no login para e-mail inexistente (#24)
- [ ] Senha: máximo de 72 bytes com 422 em vez de 500; recusar senhas óbvias (#3, #5)
- [ ] Tamanho máximo para os campos de texto (#3)
- [ ] `APP_ENV=production` recusa `JWT_SECRET` e `ADMIN_PASSWORD` padrão (#1, #5)

Prioridade média:
- [ ] Middleware de headers de segurança (#21)
- [ ] `ReadTimeout`, `WriteTimeout` e `IdleTimeout` no servidor (#22)
- [ ] Postgres do compose publicado só em `127.0.0.1` (#25)
- [ ] Log estruturado de eventos de segurança (falha de login, 401, 403, 429) (#26)

## Etapa 2 — Eventos e rastreio público

- [ ] Responder as perguntas em aberto do [PRD](PRD.md)
- [ ] Tabela `delivery_events` e regras de transição de status
- [ ] `GET /me/deliveries` para o motorista, filtrado na query (#6)
- [ ] `POST /deliveries/{id}/events` com checagem de dono e 409 para transição inválida (#6, #9)
- [ ] `GET /public/tracking/{code}` sem dados pessoais e com rate limit próprio (#15, #16, #27)
- [ ] Testes de IDOR: motorista A tentando acessar entrega do motorista B (#6)
- [ ] Testes de integração com Postgres real
- [ ] `Idempotency-Key` no `POST /deliveries` (#9)

## Etapa 3 — Front-end (React + TypeScript)

- [ ] Definir paleta, tipografia e componentes em [DESIGN.md](DESIGN.md)
- [ ] Access token curto + refresh token rotativo em cookie `HttpOnly` (#10, #14, #18)
- [ ] CORS com lista de origens por variável de ambiente (#19)
- [ ] Painel admin
- [ ] App do motorista (mobile first)
- [ ] Página pública de rastreio
- [ ] Validação nos formulários e bloqueio durante envio (#2, #9)
- [ ] CSP e regras contra XSS (#20)

## Etapa 4 — CI (GitHub Actions)

- [ ] Testes, lint e build em todo PR
- [ ] `govulncheck`, gitleaks e Trivy bloqueando merge (#1, #13)
- [ ] Dependabot para Go, Docker e Actions (#13)

## Etapa 5 — Tempo real

- [ ] WebSocket para rastreio público e painel
- [ ] Redis para pub/sub entre instâncias e rate limit compartilhado (#15)

## Etapa 6 — Notificações

- [ ] RabbitMQ e worker de e-mail quando o status muda
- [ ] Política de retenção e anonimização de dados (#27)

## Etapa 7 — Deploy na AWS

- [ ] Infra (banco em sub-rede privada, segredos no Secrets Manager/SSM, HTTPS) (#1, #16, #25)
- [ ] Usuário do banco com privilégio mínimo e backups criptografados (#25)
- [ ] Alertas de falhas de login e erros 5xx (#26)
