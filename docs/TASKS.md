# Tarefas

Backlog em ordem. Só se trabalha na etapa atual; o que surgir no caminho entra aqui antes de ser feito ([RULES.md](RULES.md#1-escopo)). Os números `#N` apontam para os itens de [SECURITY.md](SECURITY.md).

**Etapa atual: 7 — Deploy na AWS**

## Etapa 1 — Base da API ✅

- [x] Estrutura do projeto (chi, pgx, sqlc, golang-migrate, Viper)
- [x] Login com JWT e papéis admin/motorista
- [x] Cadastro e listagem de motoristas
- [x] CRUD de entregas com código de rastreio
- [x] OpenAPI servido em `/openapi.yaml`
- [x] Docker Compose com Postgres e porta configurável (`DB_PORT`)
- [x] Hot reload com air (no Docker e direto na máquina)
- [x] Documentação em `docs/`

## Etapa 1.5 — Segurança da base ✅

- [x] Só confiar em `X-Forwarded-For` com `TRUST_PROXY=true` (atrás do load balancer) (#23)
- [x] Rate limit global por IP e específico no login, com 429 e `Retry-After` (#8, #15)
- [x] Bloqueio progressivo por e-mail depois de 5 senhas erradas (#8)
- [x] Tempo constante no login para e-mail inexistente (#24)
- [x] Senha: mínimo 10, máximo 72 bytes com 422 em vez de 500, recusa de senhas comuns (#3, #5)
- [x] Tamanho máximo para os campos de texto (#3)
- [x] `APP_ENV=production` recusa `JWT_SECRET` e `ADMIN_PASSWORD` padrão e CORS sem https (#1, #5)
- [x] CORS com lista de origens em `CORS_ORIGINS` (#19)
- [x] Middleware de headers de segurança (#21)
- [x] `ReadTimeout`, `WriteTimeout` e `IdleTimeout` no servidor (#22)
- [x] Postgres do compose publicado só em `127.0.0.1` (#25)
- [x] Falha de login no log, sem o e-mail em claro (#26)
- [x] Logar também 401, 403 e 429 (#26)

## Etapa 2 — Eventos e rastreio público ✅

- [x] Responder as perguntas em aberto do [PRD](PRD.md)
- [x] Tabela `delivery_events` e regras de transição de status (com histórico das entregas existentes)
- [x] `GET /me/deliveries` para o motorista, filtrado na query (#6)
- [x] `POST /deliveries/{id}/events` com checagem de dono, 409 para transição inválida e concorrência otimista (#6, #9)
- [x] `GET /deliveries/{id}/events` com o histórico, para admin e motorista dono
- [x] `GET /public/tracking/{code}` sem dados pessoais, expirando 30 dias após a conclusão e com rate limit de 30/min (#15, #16, #27)
- [x] Testes de IDOR: motorista A tentando acessar entrega do motorista B (#6)
- [x] Testes de integração com Postgres real (testcontainers)
- [x] `Idempotency-Key` no `POST /deliveries` (#9)

## Etapa 3 — Front-end (React + TypeScript) ✅

- [x] Base do front em `web/` (Vite, React, TypeScript, Tailwind, Vitest), no `docker compose up` e na CI
- [x] Definir paleta, tipografia e componentes em [DESIGN.md](DESIGN.md)
- [x] Access token curto + refresh token rotativo em cookie `HttpOnly` na API (#10, #14, #18)
- [x] Front renova o access token pelo `/auth/refresh` e o guarda só em memória (#14, #20)
- [x] CORS com lista de origens por variável de ambiente, com `credentials` para o cookie (#19)
- [x] Painel admin: login, entregas (filtro, paginação, criação, edição, troca de status) e motoristas
- [x] App do motorista (mobile first): lista das entregas, mapa e troca de status com um toque
- [x] Página pública de rastreio
- [x] Validação nos formulários e bloqueio durante envio (#2, #9)
- [x] CSP e regras contra XSS (#20)

## Etapa 4 — CI (GitHub Actions) ✅

- [x] Testes (unitários e integração), golangci-lint, sqlc em dia e build da imagem em todo PR e push na `develop` e `main`
- [x] `govulncheck` e gitleaks (#1, #13)
- [x] Título do PR em Conventional Commits
- [x] Dependabot para Go, npm, Docker e Actions, apontando para a `develop` (#13)
- [x] Template de PR, `.editorconfig` e licença MIT
- [x] Marcar os checks como obrigatórios no ruleset (feito no GitHub, não no código)
- [x] Varredura da imagem Docker (Trivy)
- [x] Marcar o check do front como obrigatório no ruleset
- [x] Badge da CI no README

## Etapa 5 — Tempo real ✅

- [x] WebSocket para rastreio público e painel
- [x] Redis para pub/sub entre instâncias e rate limit compartilhado (#15)

## Etapa 6 — Notificações ✅

- [x] RabbitMQ e worker de e-mail quando o status muda (outbox, retry e fila de falhas)
- [x] Notificação no celular por Web Push (PWA instalável), como segundo consumidor do mesmo evento
- [x] Política de retenção e anonimização de dados (#27)

## Etapa 6.5 — Produto (várias transportadoras) ✅

O "admin" vira a transportadora, e o projeto passa a funcionar como um produto que várias transportadoras usam ([PRD](PRD.md) RF13 a RF17).

- [x] Transportadora como tenant: tabela `carriers`, `carrier_id` em usuários e entregas, papel `admin` vira `carrier` (#6)
- [x] Todas as rotas autenticadas filtram pela transportadora do token; o que é de outra responde 404 (#6)
- [x] `POST /auth/signup`: cadastro da transportadora com CNPJ opcional (inclusive o alfanumérico), já logando
- [x] `GET /me` com o usuário e a transportadora, `GET /summary` com os números do painel
- [x] WebSocket do painel por transportadora e nome da transportadora no rastreio público
- [x] Página inicial com as três entradas (transportadora, motorista, rastreio)
- [x] Cadastro e login da transportadora, login do motorista, cada um na sua tela
- [x] Painel da transportadora com visão geral, primeiros passos e o nome da transportadora no cabeçalho

## Etapa 6.6 — Endereço e rota do motorista ✅

Rota do dia como nos apps de entrega grandes ([PRD](PRD.md) RF18 a RF21), sem serviço pago.

- [x] Endereço em partes, telefone do destinatário, ponto de referência e coordenadas na entrega (API)
- [x] Retenção apaga também telefone, endereço em partes e coordenadas (#27)
- [x] Rota do dia do motorista: bipar por código ou link do QR-code, pacote sem motorista passa a ser dele (#6)
- [x] Paradas por endereço, numeração 1..N, ordem sugerida (vizinho mais próximo + 2-opt) e ordem livre
- [x] Formulário de entrega com CEP (ViaCEP) e pino no mapa (OpenStreetMap)
- [x] Etiqueta com QR-code para imprimir
- [x] Tela da rota no app do motorista: leitor de QR-code, mapa com as paradas, arrastar para reordenar

## Etapa 6.7 — Acabamento e revisão geral ✅

O que sobrou do backlog antes do deploy, mais o que a revisão geral do código encontrou.

- [x] Busca de entrega por código, nome ou e-mail do destinatário no painel, sem diferenciar acentos ([PRD](PRD.md) RF22)
- [x] Log de cada requisição em JSON, com 401, 403 e 429 como aviso e o motivo, sem código de rastreio nem query string (#26)
- [x] Login e cadastro conferem o `Origin`, contra login CSRF (#10)
- [x] Rota inexistente e método errado respondem em JSON; cliente que desistiu recebe 499 e consulta lenta, 504 (#12)
- [x] Senha montada a partir do e-mail é recusada (#5)
- [x] Limites de texto contam caracteres, não bytes; caractere nulo no JSON responde 400 (#3)
- [x] Rate limit e limite de WebSocket contam a rede `/64` inteira no IPv6 (#15, #22)
- [x] Entrega anonimizada não aceita alteração e o rastreio dela responde 404; entrega parada há mais de 1 ano também é anonimizada (#27)
- [x] Retenção apaga também as inscrições de push, as chaves de idempotência velhas e os refresh tokens vencidos (#27)
- [x] Web Push aceita o JSON que o navegador manda (`expirationTime`)
- [x] Worker descarta aviso com mais de 1 hora e esvazia a fila de push quando não tem as chaves VAPID
- [x] Rota do motorista: leitura dupla do mesmo QR-code, pacote repetido com a rota cheia, endereços sem número e pacote passado para outro motorista
- [x] OpenAPI válida e conferida contra as rotas num teste
- [x] Motorista abre a própria entrega por `GET /deliveries/{id}` e vê primeiro o que falta fazer; lista do painel filtra por motorista (#6)
- [x] App do motorista: arrastar paradas no celular, "Entreguei" atualiza a rota e alvos de toque com 44 px
- [x] Painel: pino antigo sai quando o endereço muda, etiqueta só imprime pronta e paginação sem pular página
- [x] Sessão no front: trava entre abas na renovação, sair só com a confirmação da API e cache limpo ao trocar de conta (#14)
- [x] Mapa e leitor de QR-code que não baixam não derrubam a tela; erro inesperado mostra uma tela para recarregar
- [x] Acessibilidade: pino pelo teclado, erros anunciados ao leitor de tela e mensagens da API traduzidas
- [x] Rastreio público não chama `/auth/refresh` (#15, #26)
- [x] README e documentação revisados, com telas do sistema

## Etapa 7 — Deploy na AWS

- [ ] Cabeçalho `frame-ancestors 'none'` na CDN do front (#20)
- [ ] Infra (banco em sub-rede privada, segredos no Secrets Manager/SSM, HTTPS) (#1, #16, #25)
- [ ] Usuário do banco com privilégio mínimo e backups criptografados (#25)
- [ ] Alertas de falhas de login e erros 5xx (#26)

## Sem etapa

Ideias que surgiram no caminho e ainda não têm lugar.

- [ ] Dados de exemplo para a conta demo (motoristas, entregas com coordenadas e uma rota), para quem avalia o projeto começar com o painel cheio
- [ ] No log, separar o `/auth/refresh` sem cookie (visitante da página inicial) do refresh com token inválido ou reusado, antes dos alertas (#26)
- [ ] Auditoria da edição de entregas (`PATCH`), com quem mudou o quê (#26)
- [ ] Decidir se `/openapi.yaml` fica público em produção (#12)
- [ ] Revogar as sessões ao trocar a senha, quando houver troca de senha (#14)
