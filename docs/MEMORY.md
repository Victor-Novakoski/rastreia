# Memória do projeto

Decisões tomadas e o porquê, para não rediscutir. A mais nova fica no topo.

| Data | Decisão | Por quê |
| --- | --- | --- |
| 2026-10-01 | Documentação em `docs/`: PRD, ARCHITECTURE, RULES, DESIGN, TASKS, SECURITY e esta memória. | Manter o projeto no eixo e deixar claro para quem lê o repositório. |
| 2026-10-01 | `docker compose up` sobe o ambiente de desenvolvimento com air, via `docker-compose.override.yml`. Produção: `docker compose -f docker-compose.yml up`. | É o fluxo que já uso; o Makefile fica só com atalhos opcionais. |
| 2026-10-01 | Porta do Postgres configurável por `DB_PORT`. | Quem já tem Postgres instalado na 5432 consegue subir sem mexer em arquivo. |
| 2026-10-01 | Deploy primeiro numa EC2 com Docker Compose; migração para ECS como etapa extra. | Fica no free tier, é simples de entender e ainda rende uma história de evolução. |
| 2026-10-01 | Status não muda pelo `PATCH /deliveries/{id}`, só por eventos. | Cada mudança precisa de histórico, notificação e tempo real. |
| 2026-10-01 | sqlc + pgx em vez de ORM. | SQL explícito, tipos gerados e erro de SQL aparece ao gerar, não em produção. |
| 2026-10-01 | Código de rastreio `RS` + 10 caracteres sem 0, O, 1 e I. | Difícil de adivinhar e fácil de ditar por telefone. |
| 2026-10-01 | Go 1.26. | Versão exigida pelas dependências atuais. |
| 2026-10-01 | Projeto: rastreio de entregas com Go, React, Postgres, Redis, RabbitMQ e AWS. | Mostra back-end, front-end, mensageria, testes e cloud num escopo que dá para terminar. |

## Convenções

- Commits em português.
- Usuário de teste local: `admin@rastreia.dev` / `admin12345`.
