# Tarefas

Etapas na ordem. Cada etapa termina com testes passando, README atualizado e código na `main`.

## Etapa 1: Base da API ✅

- [x] Estrutura do projeto, Docker Compose com Postgres, migrations
- [x] Login com JWT e middleware de papéis
- [x] Cadastro e listagem de motoristas; admin inicial
- [x] CRUD de entregas com código de rastreio
- [x] Testes unitários e de handler
- [x] OpenAPI
- [x] Hot reload com air no Docker

## Segurança: itens da API atual (próximo)

- [ ] Rate limit global por IP e mais rígido no login (SECURITY 8, 15)
- [ ] Bloqueio temporário por e-mail após falhas de login (SECURITY 8)
- [ ] Cabeçalhos de segurança (SECURITY 20)
- [ ] CORS com lista de origens por variável de ambiente (SECURITY 19)
- [ ] Senha mínima de 10 caracteres e lista de senhas comuns (SECURITY 5)
- [ ] Recusar a senha padrão do admin em produção (SECURITY 5)

## Etapa 2: Eventos de status

- [ ] Tabela `delivery_events` e transições de status válidas
- [ ] `POST /deliveries/{id}/events` só para o motorista da entrega (SECURITY 6)
- [ ] `GET /drivers/me/deliveries`
- [ ] `GET /track/{codigo}` público, com dados mínimos e rate limit (SECURITY 16, 25)
- [ ] Testes de integração com testcontainers

## Etapa 3: Front-end

- [ ] React + TypeScript + Vite + Tailwind + TanStack Query
- [ ] Login, Entregas, Motoristas, Minhas entregas, Rastreio público (ver DESIGN)
- [ ] Validação com zod e botões bloqueados durante envio (SECURITY 2, 9)
- [ ] `Idempotency-Key` no `POST /deliveries` (SECURITY 9)
- [ ] Testes com Vitest e Testing Library

## Etapa 4: CI

- [ ] GitHub Actions: gofmt, vet, testes, build, lint e testes do front
- [ ] govulncheck, npm audit, gitleaks e Dependabot (SECURITY 1, 13)
- [ ] Badge no README

## Etapa 5: Tempo real

- [ ] Redis pub/sub dos eventos
- [ ] WebSocket `/track/{codigo}/live` com checagem de Origin e limites (SECURITY 23, 24)

## Etapa 6: Fila de notificações

- [ ] Outbox no Postgres, RabbitMQ e worker em Go
- [ ] E-mail a cada mudança de status, gravando em `notifications`
- [ ] Retentativa e fila de mensagens com erro

## Refresh token (entre as etapas 3 e 6)

- [ ] Access token de 15 minutos e refresh token rotativo em cookie seguro, revogável (SECURITY 10, 14, 18)

## Etapa 7: Deploy na AWS

- [ ] EC2 com Docker Compose, RDS, S3 + CloudFront, HTTPS
- [ ] Segredos no SSM, banco em subnet privada, backups (SECURITY 1, 26, 27, 29)
- [ ] Deploy automático pela CI e alerta de cobrança

## Etapa 8: Acabamento

- [ ] README em português e inglês, GIF da demo, diagrama
- [ ] Usuário de teste para recrutadores

## Etapa 9: Extra

- [ ] Migrar da EC2 para ECS Fargate

## Depois (fora da v1)

Multi-empresa, mapa com GPS, app nativo, frete, relatórios, WhatsApp.
