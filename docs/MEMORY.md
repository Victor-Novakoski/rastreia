# Memória do projeto

Contexto que não está óbvio no código: decisões, o motivo de cada uma e armadilhas já encontradas. Leia antes de começar uma tarefa; atualize ao tomar uma decisão.

## Estado atual

- **Etapa:** 1, 1.5 e 2 concluídas; próxima é a 3 (front-end). Ver [TASKS.md](TASKS.md).
- **Referência de produto:** apps de entrega como Loggi e Envio Extra, dentro do escopo do [PRD](PRD.md).
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
- **2026-10-01 — `failed` pode voltar para `in_transit`.** *Nova tentativa é comum em entrega; criar outra entrega quebraria o histórico e o link do cliente.*
- **2026-10-01 — Sem foto de comprovante na v1.** *Evita upload (e seus riscos) até o fluxo principal estar pronto.*
- **2026-10-01 — Link público expira 30 dias depois de concluída a entrega.** *Menos dado pessoal exposto (LGPD) sem atrapalhar o cliente.*
- **2026-10-01 — Rate limit e bloqueio de login em memória.** *Uma instância só por enquanto; vão para o Redis quando houver mais de uma.* (httprate com Redis desde já)
- **2026-10-01 — `TRUST_PROXY` liga/desliga a leitura de `X-Forwarded-For`.** *Mais simples que uma lista de proxies; em produção só há o load balancer na frente.* (`TRUSTED_PROXIES` com faixas de IP)
- **2026-10-01 — Bloqueio de login conta e-mails inexistentes também.** *Senão o bloqueio revelaria quais e-mails têm conta.*
- **2026-10-01 — Concorrência otimista na troca de status.** *O `UPDATE` confere o status anterior (`WHERE status = from_status`); dois eventos simultâneos não se sobrepõem e o perdedor recebe 409, sem lock explícito.* (`SELECT ... FOR UPDATE`)
- **2026-10-01 — Criar entrega já grava o evento `pending`.** *O histórico começa na criação, com quem criou; entregas antigas ganharam um evento na migration (com `created_by` nulo, de sistema).*
- **2026-10-01 — `failed` exige observação; `delivered` é final.** *O motivo da falha é o que a transportadora precisa para agir; voltar de `delivered` seria corrigir dado, não um evento.*
- **2026-10-01 — Rastreio público sem as observações dos eventos.** *A observação é texto livre do motorista e pode ter dado pessoal ("deixei com o vizinho do 32"); o público vê só status e horário.*
- **2026-10-01 — Rota pública em `/public/tracking/{code}`.** *Segue a convenção `/public/...` do [DESIGN.md](DESIGN.md); código malformado, inexistente e expirado respondem o mesmo 404.*
- **2026-10-01 — `Idempotency-Key` guarda o id da entrega, não a resposta.** *O reenvio devolve a entrega como está agora; mais simples que guardar o corpo, e a chave é reservada na mesma transação da criação, então pedidos simultâneos criam uma entrega só.* (guardar status + corpo da resposta)
- **2026-10-01 — Testes de integração com testcontainers, no mesmo `go test`.** *Um container por pacote e um banco novo por teste; pulados com `-short` ou sem Docker, então `make test-short` roda sem Docker.* (build tag `integration`)
- **2026-10-01 — sqlc pela imagem Docker oficial (`make sqlc`).** *Não exige instalar o sqlc (que precisa de cgo) na máquina; a versão fica fixa (1.31.1).*
- **2026-10-01 — Documentação de produto em `docs/`.** PRD, ARCHITECTURE, RULES, DESIGN, TASKS, MEMORY e SECURITY, para o projeto não fugir do escopo.

## Armadilhas conhecidas

- **Porta 5432 ocupada** por um Postgres instalado na máquina: definir `DB_PORT=5433` no `.env` (o docker compose lê o `.env`) e ajustar a porta em `DATABASE_URL` para rodar o air fora do Docker.
- **air antigo (v1.51):** não aceita `tmp_dir` absoluto nem `build.entrypoint`. Por isso o `.air.toml` usa `tmp/` e `build.bin`, e o container troca os caminhos por flags no `CMD` do estágio `dev`. O aviso "build.bin is deprecated" nas versões novas é esperado.
- **Imagens de dev e produção** têm nomes diferentes (`rastreia-api-dev` e `rastreia-api`); se tivessem o mesmo, um `up` sem `--build` podia usar a imagem errada.
- **air no container e na máquina usam a porta 8080:** rodar um de cada vez.
- **air no container aplica migrations novas na hora:** salvar um arquivo `.sql` em `migrations/` recompila e sobe a API, que migra o banco de desenvolvimento. Escreva a migration inteira antes de salvar, ou pare o container.
- **`nullable timestamptz` no sqlc** vira `pgtype.Timestamptz` sem o override para `*time.Time` que está no `sqlc.yaml`.
- **bcrypt aceita no máximo 72 bytes de senha** e devolve erro acima disso; a validação recusa antes com 422.
- **Senha do admin de teste (`admin12345`) está na lista de senhas comuns,** mas é aceita só para o admin criado pela configuração, porque em produção a API já recusa esse valor.

## Glossário

- **Admin:** operador da transportadora.
- **Motorista (driver):** entregador; só vê as próprias entregas.
- **Destinatário:** quem recebe; não tem conta, acompanha pelo código de rastreio.
- **Evento:** registro de mudança de status de uma entrega.
