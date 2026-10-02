# Memória do projeto

Contexto que não está óbvio no código: decisões, o motivo de cada uma e armadilhas já encontradas. Leia antes de começar uma tarefa; atualize ao tomar uma decisão.

## Estado atual

- **Etapa:** 6.6 (endereço e rota do motorista) em andamento. Depois: deploy na AWS (etapa 7). Ver [TASKS.md](TASKS.md).
- **Referência de produto:** apps de entrega como Loggi e Envio Extra, dentro do escopo do [PRD](PRD.md).
- **Atualizado em:** 02/10/2026.

## Decisões

Formato: data — decisão. *Por quê.* (alternativas descartadas)

- **2026-10-02 — Endereço em colunas (CEP, rua, número...) e `address` montado pela API.** *O CEP preenche o resto, a rota agrupa pelo endereço e a etiqueta sai certa; `address` continua como a linha inteira para quem só quer ler, e entregas antigas ficam só com ela.* (JSON numa coluna, que o sqlc e as queries tratam pior; tabela de endereços, sem reuso que justifique)
- **2026-10-02 — Coordenadas vêm do front, não de geocodificação na API.** *O formulário mostra o pino no mapa e a transportadora arrasta se cair errado; a API não chama serviço externo e os testes não dependem de rede.* (Nominatim na API ao criar, que exige fila para respeitar 1 req/s e não deixa corrigir)
- **2026-10-02 — Mudar o endereço sem mandar coordenadas apaga as antigas.** *Um pino velho num endereço novo leva o motorista ao lugar errado; sem pino a parada vai para o fim da rota.*
- **2026-10-02 — Rota é do motorista e do dia (fuso de São Paulo), montada ao bipar.** *É o fluxo de quem carrega o carro na base; amanhã começa vazia.* (rota montada pela transportadora, que exige tela de despacho)
- **2026-10-02 — Bipar pacote sem motorista atribui ao motorista que bipou.** *Na base o motorista pega os pacotes da pilha; esperar a transportadora atribuir travaria a saída. De outro motorista dá 409, de outra transportadora 404.*
- **2026-10-02 — Parada = mesmo CEP, rua e número, sem olhar o complemento.** *Apartamentos de um prédio são uma parada só; entregas antigas agrupam pelo texto do endereço.*
- **2026-10-02 — Ordem sugerida por vizinho mais próximo + 2-opt em linha reta, no Go.** *Para 30 a 50 paradas fica perto do ótimo em microssegundos, sem serviço de rotas pago nem OSRM para hospedar.* (OSRM, Google/Mapbox: custo ou mais um serviço)
- **2026-10-02 — A ordem é guardada por pacote; as paradas são calculadas na leitura.** *Pacotes da mesma parada ficam juntos no lugar do primeiro, então reordenar paradas é só mandar os ids na nova ordem.*

- **2026-10-02 — Várias transportadoras, cada uma um tenant, com `carrier_id` em usuários e entregas.** *O "admin" único não fazia sentido como produto; com o tenant, uma transportadora se cadastra e começa a usar sozinha.* (banco ou schema por transportadora, pesado demais para o tamanho do projeto; Row Level Security do Postgres, que esconde a regra fora do código e complica os testes)
- **2026-10-02 — `carrier_id` no token (`cid`) e filtro no service, não em middleware.** *O token já diz quem é e de onde; os services recebem o `auth.Claims` e cada query filtra ou confere a transportadora, o que os testes de IDOR cobrem.* (buscar a transportadora no banco a cada requisição)
- **2026-10-02 — Papel `admin` vira `carrier`.** *O nome bate com o produto; a migration converte os usuários antigos, e tokens antigos (sem `cid`) são recusados, então quem estava logado entra de novo.*
- **2026-10-02 — Dados antigos vão para "Minha transportadora".** *A migration cria uma transportadora só se já existirem usuários ou entregas, então um banco novo começa vazio.*
- **2026-10-02 — Cadastro da transportadora e do responsável num único `INSERT` com CTE.** *É atômico sem precisar de transação no service: e-mail repetido não deixa transportadora sem dono.* (transação com `InTx` no pacote user)
- **2026-10-02 — CNPJ opcional, guardado sem pontuação e aceitando o formato alfanumérico.** *A Receita passou a emitir CNPJ com letras em julho de 2026; o dígito verificador usa o código ASCII menos 48, que dá o mesmo resultado para os números.* (CNPJ obrigatório, que atrapalha quem só quer testar)
- **2026-10-02 — Um usuário por transportadora por enquanto.** *O responsável cadastra a conta e os motoristas; convites e papéis internos ficam fora do escopo* ([PRD](PRD.md)).

- **2026-10-02 — Anonimizar 90 dias depois de concluída, sem apagar a entrega.** *O link público já expira em 30; a transportadora ainda precisa de alguns meses para reclamações, e os relatórios continuam contando entregas e status. Roda na própria API a cada hora: o `UPDATE` só pega linhas não anonimizadas, então várias instâncias podem rodar juntas.* (apagar a linha, que quebra relatórios; job separado ou cron, mais uma peça para subir)
- **2026-10-02 — `delivery_events` como outbox, com um relay na API que publica no RabbitMQ.** *Publicar direto depois do commit perde a notificação se a API cair ou o RabbitMQ estiver fora; com o outbox o evento e a notificação são gravados juntos. `SKIP LOCKED` deixa várias instâncias rodarem o relay.* (publicar após o commit; tabela `outbox` separada, que duplicaria o evento)
- **2026-10-02 — Retry com fila de espera (TTL de 30s) e fila de falhas depois de 5 tentativas.** *O RabbitMQ conta as tentativas no header `x-death`, sem estado no worker; mensagem inválida vai direto para a fila de falhas.* (requeue imediato, que vira loop; plugin de delayed message)
- **2026-10-02 — Pelo menos uma vez, sem deduplicar no worker.** *Duplicar um e-mail raro é aceitável e deduplicar exigiria banco ou Redis no worker.*
- **2026-10-02 — Worker sem acesso ao banco; a mensagem leva nome e e-mail.** *Menos permissão no worker.* (worker buscando a entrega pelo id)
- **2026-10-02 — E-mail com go-mail e Mailpit em desenvolvimento.** *go-mail cuida de MIME, UTF-8 no assunto e STARTTLS; o Mailpit mostra os e-mails sem mandar nada de verdade.* (`net/smtp`, MailHog, que parou de ser mantido)
- **2026-10-02 — Celular por Web Push (PWA), não app nativo.** *Avisa como um app sem loja nem segundo código; no iPhone só funciona com o site adicionado à tela de início.* (React Native, SMS)
- **2026-10-02 — Inscrição de push pertence à entrega, não a uma pessoa.** *O destinatário não tem conta; quem tem o código pode pedir aviso, como já pode ver o rastreio. Limite de 10 navegadores por entrega e apagadas quando a entrega é entregue.*
- **2026-10-02 — Endpoint de push só em hosts dos serviços dos navegadores.** *O worker faz POST na URL que o navegador manda; sem a lista, seria SSRF ([SECURITY.md](SECURITY.md) #17).* (bloquear só IPs privados, que exige checar depois do DNS)
- **2026-10-02 — Worker do push lê o banco.** *As inscrições mudam a qualquer hora; mandar todas dentro da mensagem do evento deixaria a fila com dados velhos.* O e-mail continua sem banco.
- **2026-10-02 — Service worker registrado só ao ativar os avisos.** *Quem não usa push não ganha um service worker; o front não tem cache offline.* (vite-plugin-pwa, que traria cache e mais configuração)

- **2026-10-02 — Tempo real por WebSocket (`coder/websocket`), servidor só envia.** *O rastreio público recebe o mesmo corpo do `GET`, então a página não refaz a consulta; o painel recebe só `{delivery_id, status}` e invalida o cache do TanStack Query, sem dado pessoal no fio.* (SSE, que serviria e é mais simples, mas o PRD pede WebSocket; polling)
- **2026-10-02 — Token do painel na primeira mensagem do WebSocket, não na URL.** *O navegador não manda `Authorization` no WebSocket, e token na URL vai parar em log. A API fecha com o código 4001 quando o token vence, e o front renova e reconecta.* (cookie, que exigiria mais checagem de CSRF; subprotocolo)
- **2026-10-02 — Redis opcional (`REDIS_URL`): com ele, tempo real, rate limit e bloqueio de login valem para todas as instâncias; sem ele, ficam em memória.** *Rodar a API sozinha (`make run`, testes) continua simples, e o compose sobe o Redis.* (Redis obrigatório)
- **2026-10-02 — Uma inscrição por padrão (`PSUBSCRIBE rastreia:live:*`) por instância, que repassa para o broker em memória.** *Uma conexão no Redis por instância, não uma por navegador.* (um `SUBSCRIBE` por WebSocket)
- **2026-10-02 — Sem o Redis, o login responde 503; o rate limit cai para memória.** *Sem contador não há proteção contra força bruta, então o login espera; o rate limit em memória ainda protege cada instância.* (deixar logar sem contador)

- **2026-10-02 — CSP do front numa `<meta>` injetada no build.** *O front vai ser estático numa CDN, e a política acompanha o HTML sem depender da configuração do servidor; só `frame-ancestors` precisa ir no cabeçalho.* (CSP só no cabeçalho da CDN; nonce, que exige servidor)

- **2026-10-02 — App do motorista lê até 100 entregas de `/me/deliveries` e acha a entrega na lista.** *A API não tem `GET /me/deliveries/{id}`, e 100 cobre o dia de um motorista; entregas antigas somem da lista, o que não atrapalha.* (rota nova na API)
- **2026-10-02 — No app do motorista, 409 de uma repetição conta como sucesso se o status já é o pedido.** *Com sinal ruim, o primeiro envio pode chegar e a resposta não; o motorista toca de novo e não deve ver erro.*

- **2026-10-02 — TanStack Query no painel.** *Lista, detalhe e formulários compartilham dados e precisam invalidar o cache depois de cada mudança; escrever isso à mão seria mais código e mais bug.* (fetch com useEffect)
- **2026-10-02 — Access token só em memória, renovado no 401.** *Recarregar a página chama `/auth/refresh` com o cookie; um 401 renova uma vez e repete a chamada. Os refreshes simultâneos dividem a mesma requisição, porque usar o mesmo refresh token duas vezes derruba a sessão.* (`localStorage`, que o XSS lê; renovar por timer)
- **2026-10-02 — Rotas do front em português: `/`, `/transportadora/entrar`, `/transportadora/cadastro`, `/transportadora` (visão geral, `entregas`, `motoristas`), `/motorista/entrar`, `/motorista`, `/rastreio`.** *São as URLs que o usuário vê; cada público tem a própria porta de entrada. `/entrar` antigo leva para a página inicial.*
- **2026-10-02 — Página inicial com uma amostra do rastreio feita em HTML, não imagem.** *Mostra o produto sem screenshot para manter atualizado e abre rápido.*

- **2026-10-02 — Refresh token opaco no banco (hash SHA-256), com rotação e família.** *Dá para revogar (logout, reuso) e o banco vazado não entrega tokens usáveis; reuso derruba a família inteira.* (refresh em JWT, sem estado: não dá para revogar)
- **2026-10-02 — Sem período de tolerância para refresh simultâneo.** *Mais simples e mais seguro; o front garante uma renovação por vez. Duas abas renovando ao mesmo tempo podem derrubar a sessão: se virar problema, entra uma tolerância de poucos segundos.*
- **2026-10-02 — Cookie `rastreia_refresh` com `Path=/auth` e `SameSite=Strict`, mais checagem de `Origin`.** *O cookie só vai para as rotas de sessão, e o `Origin` fecha o CSRF. Exige front e API no mesmo site.* (prefixo `__Host-`, que obriga `Path=/`)
- **2026-10-02 — Front em `web/`, no mesmo repositório.** *Um PR muda API e tela juntos, e a CI confere os dois.* (repositório separado)
- **2026-10-02 — Vite + React + TypeScript + Tailwind, com oxlint e Vitest.** *Stack padrão do mercado e rápida; oxlint veio no template do Vite e substitui o ESLint.* (Next.js: sem SSR necessário, o front é estático numa CDN)
- **2026-10-02 — React Router, sem biblioteca de estado ou de requisições por enquanto.** *Uma página pública só pede um `fetch`; TanStack Query entra se o painel precisar de cache.* (RULES: nada "para o futuro")
- **2026-10-02 — Fonte do sistema e ícones SVG próprios.** *Página pública abre rápido em rede ruim e não depende de CDN externa, o que facilita a CSP.* (Inter via Google Fonts, lucide-react)
- **2026-10-02 — Front chama a API direto pela `VITE_API_URL`.** *O CORS já aceita `http://localhost:5173`; em produção o front fica na CDN e a API em outro domínio.* (proxy do Vite)

- **2026-10-01 — Conventional Commits e CI no GitHub Actions.** *Histórico legível e todo PR conferido (lint, sqlc, testes, imagem, govulncheck, gitleaks, título do PR).* Actions fixadas por hash do commit.
- **2026-10-01 — IP do cliente atrás de proxy = último valor do `X-Forwarded-For`.** *O `middleware.RealIP` do chi usa o primeiro, que o cliente pode falsificar (achado pelo golangci-lint).* (chi RealIP, httprate KeyByRealIP)
- **2026-10-01 — Licença MIT.** *Padrão para portfólio: qualquer um pode ler e reutilizar com crédito.*
- **2026-10-01 — Git flow simples: `develop` + branches de feature, PR obrigatório.** *Push direto bloqueado em `main` e `develop`; branch apagada automaticamente no merge.* (push direto na main)
- **2026-10-01 — Go com chi, sem framework.** *Biblioteca padrão + roteador leve deixa o código explícito e fácil de testar.* (Gin, Echo, Fiber)
- **2026-10-01 — sqlc + pgx em vez de ORM.** *SQL escrito à mão e revisável, código tipado gerado, parâmetros sempre — elimina SQL injection por construção.* (GORM, ent)
- **2026-10-01 — Migrations embutidas e aplicadas ao subir a API.** *Um binário só, sem passo manual; o banco sempre fica na versão do código.*
- **2026-10-01 — JWT HS256 com papel no token.** *Simples para a etapa 1. Na etapa 3 passou a durar 15 min, com refresh rotativo* ([SECURITY.md](SECURITY.md) #14).
- **2026-10-01 — Status muda só por evento, nunca por PATCH.** *Garante histórico completo para o rastreio público e auditoria.*
- **2026-10-01 — Código de rastreio aleatório (`RS` + 10 caracteres sem 0/O/1/I).** *Legível por telefone e impossível de adivinhar a partir de outro código; o id sequencial nunca é público.*
- **2026-10-01 — Uma única transportadora por instalação.** *Multi-tenant fica fora de escopo para manter o foco* ([PRD](PRD.md)). Substituída em 02/10: o projeto passou a ser um produto com várias transportadoras.
- **2026-10-01 — Hot reload com air via `docker-compose.override.yml`.** *`docker compose up` já sobe o ambiente de desenvolvimento, sem make nem `-f`; a imagem de produção continua sendo o estágio final do Dockerfile.* (`make dev`, arquivo `docker-compose.dev.yml`)
- **2026-10-01 — `failed` pode voltar para `in_transit`.** *Nova tentativa é comum em entrega; criar outra entrega quebraria o histórico e o link do cliente.*
- **2026-10-01 — Sem foto de comprovante na v1.** *Evita upload (e seus riscos) até o fluxo principal estar pronto.*
- **2026-10-01 — Link público expira 30 dias depois de concluída a entrega.** *Menos dado pessoal exposto (LGPD) sem atrapalhar o cliente.*
- **2026-10-01 — Rate limit e bloqueio de login em memória.** *Uma instância só por enquanto.* Substituída em 02/10 pelo Redis opcional.
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

- **Transportadora (carrier):** empresa que usa o Rastreia; é o tenant. O papel `carrier` é de quem toca a transportadora (antes chamado de admin).
- **Motorista (driver):** entregador; só vê as próprias entregas.
- **Destinatário:** quem recebe; não tem conta, acompanha pelo código de rastreio.
- **Evento:** registro de mudança de status de uma entrega.
