# Memória do projeto

Contexto que não está óbvio no código: decisões, o motivo de cada uma e armadilhas já encontradas. Leia antes de começar uma tarefa; atualize ao tomar uma decisão.

## Estado atual

- **Etapa:** 1 e 1.5 concluídas; próxima é a 2 (eventos e rastreio público). Ver [TASKS.md](TASKS.md).
- **Atualizado em:** 01/10/2026.

## Decisões

Formato: data — decisão. *Por quê.* (alternativas descartadas)

- **2026-10-01 — Go com chi, sem framework.** *Biblioteca padrão + roteador leve deixa o código explícito e fácil de testar.* (Gin, Echo, Fiber)
- **2026-10-01 — sqlc + pgx em vez de ORM.** *SQL escrito à mão e revisável, código tipado gerado, parâmetros sempre — elimina SQL injection por construção.* (GORM, ent)
- **2026-10-01 — Migrations embutidas e aplicadas ao subir a API.** *Um binário só, sem passo manual; o banco sempre fica na versão do código.*
- **2026-10-01 — JWT HS256 com papel no token.** *Simples para a etapa 1. Será trocado por access token curto + refresh rotativo na etapa 3* ([SECURITY.md](SECURITY.md) #14).
- **2026-10-01 — Status muda só por evento, nunca por PATCH.** *Garante histórico completo para o rastreio público e auditoria.*
- **2026-10-01 — Código de rastreio aleatório (`RS` + 10 caracteres sem 0/O/1/I).** *Legível por telefone e impossível de adivinhar a partir de outro código; o id sequencial nunca é público.*
- **2026-10-01 — Uma única transportadora por instalação.** *Multi-tenant fica fora de escopo para manter o foco* ([PRD](PRD.md)).
- **2026-10-01 — Hot reload com air via `docker-compose.override.yml`.** *`docker compose up` já sobe o ambiente de desenvolvimento, sem make nem `-f`; a imagem de produção continua sendo o estágio final do Dockerfile.* (`make dev`, arquivo `docker-compose.dev.yml`)
- **2026-10-01 — Rate limit e bloqueio de login em memória.** *Uma instância só por enquanto; vão para o Redis quando houver mais de uma.* (httprate com Redis desde já)
- **2026-10-01 — `TRUST_PROXY` liga/desliga a leitura de `X-Forwarded-For`.** *Mais simples que uma lista de proxies; em produção só há o load balancer na frente.* (`TRUSTED_PROXIES` com faixas de IP)
- **2026-10-01 — Bloqueio de login conta e-mails inexistentes também.** *Senão o bloqueio revelaria quais e-mails têm conta.*
- **2026-10-01 — Documentação de produto em `docs/`.** PRD, ARCHITECTURE, RULES, DESIGN, TASKS, MEMORY e SECURITY, para o projeto não fugir do escopo.

## Armadilhas conhecidas

- **Porta 5432 ocupada** por um Postgres instalado na máquina: definir `DB_PORT=5433` no `.env` (o docker compose lê o `.env`) e ajustar a porta em `DATABASE_URL` para rodar o air fora do Docker.
- **air antigo (v1.51):** não aceita `tmp_dir` absoluto nem `build.entrypoint`. Por isso o `.air.toml` usa `tmp/` e `build.bin`, e o container troca os caminhos por flags no `CMD` do estágio `dev`. O aviso "build.bin is deprecated" nas versões novas é esperado.
- **Imagens de dev e produção** têm nomes diferentes (`rastreia-api-dev` e `rastreia-api`); se tivessem o mesmo, um `up` sem `--build` podia usar a imagem errada.
- **air no container e na máquina usam a porta 8080:** rodar um de cada vez.
- **bcrypt aceita no máximo 72 bytes de senha** e devolve erro acima disso; a validação recusa antes com 422.
- **Senha do admin de teste (`admin12345`) está na lista de senhas comuns,** mas é aceita só para o admin criado pela configuração, porque em produção a API já recusa esse valor.

## Glossário

- **Admin:** operador da transportadora.
- **Motorista (driver):** entregador; só vê as próprias entregas.
- **Destinatário:** quem recebe; não tem conta, acompanha pelo código de rastreio.
- **Evento:** registro de mudança de status de uma entrega.
