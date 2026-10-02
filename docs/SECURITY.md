# Segurança

Como o Rastreia trata cada risco, o que já está feito e o que falta. As tarefas pendentes estão em [TASKS.md](TASKS.md) e o checklist rápido para cada mudança está em [RULES.md](RULES.md#3-segurança).

Situação revisada em 02/10/2026, durante a etapa 3 (refresh token em cookie).

**Legenda:** ✅ feito · 🟡 parcial · 🔴 pendente · ⚪ ainda não se aplica (regra definida para quando se aplicar)

## Resumo

| # | Risco | Situação | Prioridade | O que é |
| --- | --- | --- | --- | --- |
| 1 | Variáveis de ambiente expostas | 🟡 | Etapas 4 e 7 | Senhas, chaves e segredos vazando por arquivo commitado, imagem Docker ou valor padrão usado em produção. |
| 2 | Validação no front-end | ✅ | — | Conferir os dados no formulário para dar retorno rápido ao usuário. Ajuda na experiência, mas não protege nada: dá para burlar. |
| 3 | Validação no back-end | ✅ | — | A API confere tipo, formato e tamanho de tudo que recebe. É a validação que realmente protege. |
| 4 | SQL Injection | ✅ | — | Texto enviado pelo usuário vira parte do comando SQL e consegue ler ou apagar dados do banco. |
| 5 | Autenticação fraca | 🟡 | Baixa | Senha fraca ou previsível, login que dá pistas, ou credencial padrão que nunca foi trocada. |
| 6 | IDOR | ✅ | — | Trocar o id na URL (ex.: /deliveries/2 por /deliveries/3) e acessar dado de outra pessoa. |
| 7 | Senhas no banco | ✅ | — | Guardar a senha como texto no banco: se o banco vazar, todas as senhas vazam junto. |
| 8 | Força bruta | ✅ | — | Tentar milhares de senhas seguidas até acertar. |
| 9 | Envio duplicado | ✅ | — | Clique duplo, rede lenta ou reenvio que cria o mesmo registro duas vezes. |
| 10 | CSRF | ✅ | — | Outro site faz o navegador do usuário logado enviar uma ação para a API sem ele perceber. |
| 11 | Upload sem validação | ⚪ | — | Arquivo malicioso disfarçado (ex.: script com extensão .jpg) ou grande demais para o servidor. |
| 12 | Vazamento de informação | 🟡 | Média | Mensagem de erro, stack trace ou cabeçalho que conta detalhes internos para um atacante. |
| 13 | Dependências vulneráveis | 🟡 | Média | Biblioteca de terceiros com falha de segurança conhecida. |
| 14 | Tokens | ✅ | — | Token que vale por muito tempo, não pode ser revogado ou carrega dados demais. |
| 15 | Rate limit | ✅ | Etapa 5 | Limitar quantas requisições cada cliente faz por minuto, contra abuso e força bruta. |
| 16 | Dados sensíveis expostos | 🟡 | Etapa 7 (HTTPS) | Resposta da API, log ou link público mostrando dado pessoal ou secreto além do necessário. |
| 17 | SSRF | ✅ | — | Fazer o servidor chamar uma URL escolhida pelo atacante, como a rede interna ou os metadados da nuvem. |
| 18 | Cookies inseguros | ✅ | — | Cookie que o JavaScript pode ler, que trafega sem HTTPS ou que é enviado por outros sites. |
| 19 | CORS | ✅ | — | Regra do navegador que diz quais sites podem chamar a API. |
| 20 | XSS | ✅ | — | Script injetado num dado (ex.: no nome) que roda no navegador de quem abre a página. |
| 21 | Headers de segurança | ✅ | — | Cabeçalhos HTTP que mandam o navegador se proteger (ex.: não abrir a página dentro de iframe, só usar HTTPS). |
| 22 | Timeouts e negação de serviço | 🟡 | Etapa 5 | Conexões lentas ou requisições enormes que prendem o servidor e derrubam a API. |
| 23 | Falsificação de IP | ✅ | — | Cliente mente o IP pelo cabeçalho X-Forwarded-For para escapar do rate limit. |
| 24 | Enumeração de e-mails | ✅ | — | Descobrir quais e-mails têm conta pela mensagem ou pelo tempo de resposta do login. |
| 25 | Banco de dados exposto | 🟡 | Etapa 7 | Banco acessível pela rede ou pela internet, sem precisar passar pela API. |
| 26 | Logs e auditoria | 🟡 | Média | Registrar quem fez o quê e os eventos suspeitos, para investigar e criar alertas. |
| 27 | LGPD e retenção de dados | ✅ | — | Lei de proteção de dados: coletar só o necessário, mostrar o mínimo e apagar quando não precisar mais. |

Os itens 1 a 19 são a lista original; os itens 20 a 27 completam a cobertura.

---

## 1. Variáveis de ambiente expostas — 🟡

**Feito**
- `.env` está no `.gitignore` e no `.dockerignore` (não entra no git nem na imagem).
- Segredos só por variável de ambiente; `.env.example` com valores de desenvolvimento.
- A API não sobe com `JWT_SECRET` com menos de 32 caracteres.
- Com `APP_ENV=production`, a API **recusa subir** com o `JWT_SECRET` ou o `ADMIN_PASSWORD` de desenvolvimento, ou com origem de CORS sem `https` (`internal/config`, com testes).

- gitleaks na CI varre todo o histórico do git a cada PR, procurando chave ou senha commitada por engano.

**Falta**
- Em produção (AWS), segredos no Secrets Manager ou SSM Parameter Store, nunca em arquivo.

## 2. Validação no front-end — ✅

Feito no painel admin: campos com `required`, `type="email"` e `maxLength` iguais aos limites da API, e os `fields` do 422 traduzidos e mostrados embaixo de cada campo (`web/src/lib/fields.ts`). Regra: o front valida para dar **feedback rápido** ao usuário (campos obrigatórios, formato de e-mail, tamanho), mas nunca é a proteção. Toda regra do front existe também no back. As mensagens de `fields` do 422 aparecem embaixo do campo.

## 3. Validação no back-end — ✅

- Validação centralizada no service com `apperr.Validator`, retornando 422 com todos os campos inválidos.
- Normalização (trim, e-mail em minúsculas) antes de validar.
- `httpx.Decode` limita o corpo a 1 MB e rejeita campos desconhecidos (impede *mass assignment*, ex.: mandar `"status"` ou `"role"` no corpo).
- Tamanho máximo nos textos: nome 120, e-mail 254, endereço 300 caracteres.
- Senha com mais de 72 bytes (limite do bcrypt) volta 422 em vez de 500.
- O banco reforça com `CHECK` (status e papel), `NOT NULL` e `UNIQUE`.

## 4. SQL Injection — ✅

Todo SQL fica em `internal/database/queries/*.sql` e o sqlc gera código com parâmetros (`$1`, `$2`). Não existe SQL montado com concatenação. Regra em [RULES.md](RULES.md): SQL só pelo sqlc. Filtros dinâmicos (ex.: status opcional) são resolvidos com `sqlc.narg`, não com string.

## 5. Autenticação fraca — 🟡

**Feito**
- Senhas com bcrypt; mínimo de 10 caracteres, máximo de 72 bytes e recusa de senhas comuns (`1234567890`, `senha12345`…).
- Mesma mensagem e mesmo tempo de resposta para e-mail inexistente e senha errada (item 24).
- JWT com algoritmo fixo (HS256), validade obrigatória e papel validado ao ler o token.
- O papel vem do banco no login, nunca do corpo da requisição.
- Em produção a API não sobe com a senha padrão do admin.

**Falta**
- Recusar senha igual ou parecida com o e-mail.
- Ver também tokens (14).

## 6. IDOR (acesso a recurso de outra pessoa pelo id) — ✅

- Cada transportadora é um tenant. O `carrier_id` vem do token, e toda leitura ou escrita confere a transportadora: listagens filtram na query SQL, buscas por id respondem **404** quando a entrega é de outra transportadora, e motorista de outra transportadora não pode ser atribuído (422). Testes: `TestCarrierIsolation` (service) e `TestIntegration_CarriersAreIsolated` (API com Postgres real).
- Rotas da transportadora (`/drivers`, `/deliveries`, `/deliveries/{id}`, `/summary`) respondem 403 para motorista.
- O motorista lista entregas por `/me/deliveries`, e o filtro `driver_id = <id do token>` fica **na query SQL** (`ListDriverDeliveries`): entrega de outro motorista nunca sai do banco.
- `GET`/`POST /deliveries/{id}/events` conferem se a entrega pertence ao motorista; se não pertencer (ou não tiver motorista), respondem **404**, igual a uma entrega inexistente, para não confirmar que o id existe.
- O rastreio público usa só o código aleatório (10 caracteres de um alfabeto de 32, cerca de 50 bits), nunca o id sequencial.
- Testes: unitários (`TestEvents_DriverOnlySeesOwnDeliveries`) e de integração pela API com Postgres real (`TestIntegration_DriversOnlyReachTheirOwnDeliveries`): motorista B tentando ler, alterar e listar entrega do motorista A.

## 7. Senhas no banco — ✅

- Só o hash bcrypt é salvo (`password_hash`), nunca a senha.
- O tipo `user.User`, que é o que a API devolve, não tem o campo do hash.
- Senha nunca vai para log.
- Futuro: se o custo do bcrypt mudar, refazer o hash no próximo login.

## 8. Ataque de força bruta — ✅

- Limite de 10 tentativas de login por minuto por IP, com 429 e `Retry-After`.
- Bloqueio por e-mail: depois de 5 senhas erradas, o e-mail fica bloqueado por 1 minuto, e o tempo dobra a cada nova falha até 15 minutos. Vale também para e-mails que não existem, para o bloqueio não revelar quem tem conta. Senha certa zera o contador.
- Cada falha é registrada no log com o IP e uma impressão do e-mail (não o e-mail em si).
- Com `REDIS_URL`, o contador fica no Redis e vale para todas as instâncias, numa operação atômica (script Lua). As chaves levam um hash SHA-256 do e-mail, nunca o e-mail. Se o Redis não responde, o login devolve 503 em vez de seguir sem proteção.

## 9. Bloquear durante envio (envio duplicado) — 🟡

**Back (feito)**
- `POST /deliveries` aceita o cabeçalho `Idempotency-Key`: a mesma chave do mesmo usuário em até 24h devolve a entrega criada na primeira vez (com `Idempotent-Replayed: true`) em vez de criar outra. A mesma chave com outro corpo responde 422.
- A chave é reservada na mesma transação que cria a entrega, então duas requisições simultâneas com a mesma chave criam uma entrega só (teste de integração com 10 requisições em paralelo).
- Eventos de status: transição repetida (ex.: `picked_up` → `picked_up`) responde 409, então reenviar não duplica histórico. Dois eventos simultâneos na mesma entrega: só um é aplicado, o outro recebe 409 (a troca de status confere o status anterior no `UPDATE`).

**Front (feito):** o `Button` fica desabilitado e com indicador enquanto a requisição não volta, e o formulário ignora um segundo envio. O formulário de nova entrega gera uma `Idempotency-Key` (`crypto.randomUUID()`), mantida se a rede falhar e trocada quando a API responde com erro.

## 10. CSRF — ✅

As rotas que alteram dados usam `Authorization: Bearer`, que o navegador não envia sozinho. Só `/auth/refresh` e `/auth/logout` dependem do cookie, e elas têm duas barreiras: o cookie é `SameSite=Strict` e restrito a `Path=/auth`, e a requisição precisa de um cabeçalho `Origin` da lista `CORS_ORIGINS` (sem `Origin`, 403).

Se algum dia a sessão inteira for por cookie, token anti-CSRF obrigatório nas rotas que alteram dados.

## 11. Upload sem validação — ⚪

Não há upload hoje. Se o comprovante de entrega com foto entrar (pergunta em aberto no [PRD](PRD.md)):
- Tamanho máximo (ex.: 5 MB) com `http.MaxBytesReader`.
- Tipo verificado pelo conteúdo do arquivo (*magic bytes*), não pela extensão nem pelo `Content-Type` enviado; só JPEG, PNG e WebP.
- Reprocessar a imagem no servidor para remover metadados (EXIF tem GPS).
- Nome gerado pelo servidor; arquivo em bucket S3 privado, entregue por URL assinada e temporária.
- Nunca servir o arquivo pelo mesmo domínio da API como HTML.

## 12. Vazamento de informação — 🟡

**Feito**
- Erro inesperado vira `500 {"error":"internal error"}`; o detalhe só vai para o log.
- Pânico é capturado pelo middleware `Recoverer`, sem stack trace na resposta.
- Login não diz se o e-mail existe (pela mensagem).

**Falta**
- O tempo de resposta do login ainda revela se o e-mail existe (item 24).
- Headers de segurança (item 21).
- Decidir se `/openapi.yaml` continua público em produção (hoje é; não expõe segredo, mas mapeia a API).

## 13. Dependências vulneráveis — ✅

**Feito**
- `govulncheck` rodado em 01/10/2026: **nenhuma vulnerabilidade alcançável pelo código**. Ele aponta o GO-2026-5932, no pacote `openpgp` de `golang.org/x/crypto`, que o projeto não usa (só usamos `bcrypt`).
- Na etapa 2 o testcontainers trouxe o `moby/go-archive` v0.2.0 com o GO-2026-6253 (alcançável só pelo código de teste); atualizado para a v0.3.0, que corrige.
- Imagem final distroless, sem shell nem gerenciador de pacotes, rodando como usuário não-root.

- `govulncheck` roda na CI em todo PR.
- Dependabot abre PR semanal para a `develop` com atualizações de módulos Go, imagens Docker e GitHub Actions.
- As GitHub Actions são fixadas pelo hash do commit, não pela tag, porque tags podem ser trocadas por quem invadir o repositório da action.
- Trivy varre a imagem final na CI e falha o PR em vulnerabilidade HIGH ou CRITICAL que já tenha correção.

## 14. Tokens mal otimizados — ✅

**Feito**
- Access token JWT de 15 min (`JWT_TTL`), com algoritmo fixo, claims mínimos (`sub`, `role`, `iat`, `exp`) e `iss`/`aud` validados.
- Refresh token opaco (32 bytes aleatórios) no cookie `rastreia_refresh`, salvo no banco **só como hash** (SHA-256), válido por `REFRESH_TTL` (7 dias) sem uso.
- Rotação: cada uso marca o token como usado e entrega o próximo da mesma família. Reuso de um token já trocado revoga a família inteira (sinal de roubo). Dois refreshes simultâneos com o mesmo token também contam como reuso.
- `POST /auth/logout` revoga a sessão.

**Falta:** revogar as sessões ao trocar a senha ou desativar o usuário, quando essas funções existirem. No front, o access token fica só em memória (item 20).

## 15. Rate limit — ✅

**Feito**
- Limite global de 120 requisições por minuto por IP e de 10 por minuto no login, com 429 e `Retry-After` (`internal/server/middleware.go`, com testes).
- Limite próprio de 30 por minuto por IP no rastreio público (`/public/tracking/{code}`), para dificultar a varredura de códigos.
- O IP usado é o da conexão, a não ser que `TRUST_PROXY=true` (item 23).
- A página de rastreio e o WebSocket dela dividem o mesmo limite.
- Com `REDIS_URL`, os contadores ficam no Redis e valem para todas as instâncias; se o Redis cair, cada instância volta a contar em memória até ele voltar.
- Em produção o Redis não deve ficar exposto na internet; no compose de desenvolvimento ele só escuta em `127.0.0.1`.

## 16. Dados sensíveis expostos — 🟡

**Feito**
- Hash de senha nunca sai da API; `.env` fora do git; logs não registram corpo de requisição nem token.
- O rastreio público tem resposta própria (`delivery.Tracking`): código, status, histórico (status e horário) e primeiro nome do destinatário. **Não** inclui e-mail, endereço, sobrenome, motorista, ids internos nem as observações dos eventos (texto livre do motorista pode ter dado pessoal). Testado no serviço e pela API.

**Falta**
- HTTPS obrigatório em produção (HSTS).
- Banco exposto (item 25).

## 17. SSRF — ✅

O único lugar que chama uma URL vinda de fora é o Web Push: o navegador entrega o `endpoint` da inscrição e o worker faz um POST nele. Feito:
- O endpoint só é aceito com `https`, sem porta nem usuário, e com o host numa lista fechada dos serviços de push dos navegadores (Google, Mozilla, Microsoft, Apple). Qualquer outro host, IP ou `localhost` responde 422 (`internal/push`, testado).
- O worker não segue redirecionamento e tem timeout de 15s.
- A chave `p256dh` precisa ser um ponto válido da curva P-256, e há no máximo 10 navegadores por entrega.

Se outra URL vinda do usuário aparecer (webhook, geocodificação, imagem por URL):
- Lista de hosts permitidos.
- Bloquear IPs privados, loopback e o endereço de metadados da nuvem (`169.254.169.254`), verificando o IP **depois** de resolver o DNS.
- Timeout curto e sem seguir redirecionamentos.
- Na AWS, IMDSv2 obrigatório.

## 18. Cookies inseguros — ✅

O único cookie é o `rastreia_refresh`: `HttpOnly`, `SameSite=Strict`, `Path=/auth`, `Secure` sempre (Chrome e Firefox aceitam cookie `Secure` em `http://localhost`, então o desenvolvimento não precisa de HTTPS) e validade igual à do refresh token. Não dá para usar o prefixo `__Host-`, que exige `Path=/`.

**Requisito de deploy:** front e API no mesmo site (ex.: `app.dominio` e `api.dominio`), senão o `SameSite=Strict` impede o envio do cookie.

## 19. CORS — ✅

- Lista exata de origens vinda de `CORS_ORIGINS` (padrão: `http://localhost:5173`, o Vite), sem `*`. Em produção só aceita `https`.
- Só os métodos (`GET`, `POST`, `PATCH`) e cabeçalhos (`Authorization`, `Content-Type`) usados.
- `credentials` liberado, porque o refresh token vai no cookie (item 18). Por isso a lista de origens nunca pode ter `*`.

## 20. XSS — ✅

- React escapa todo texto vindo da API. O oxlint barra `dangerouslySetInnerHTML` (`react/no-danger`), links `javascript:` e `target="_blank"` sem `rel="noopener noreferrer"`.
- O build injeta uma Content-Security-Policy no `index.html` (`web/csp.ts`): scripts e estilos só do próprio site, nada inline, e conexões só com a origem de `VITE_API_URL`. No `npm run dev` ela fica de fora, porque o hot reload do Vite usa script inline.
- O access token fica em memória, não em `localStorage`; o refresh token está num cookie `HttpOnly`. Um XSS não leva a sessão embora.

**Deploy (etapa 7):** `frame-ancestors` não funciona em `<meta>`, então a CDN do front precisa mandar `Content-Security-Policy: frame-ancestors 'none'` (ou `X-Frame-Options: DENY`) no cabeçalho.

## 21. Headers de segurança — ✅

Middleware em todas as respostas: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` e `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'` (contra clickjacking), `Referrer-Policy: no-referrer`, `Cache-Control: no-store` e, em produção, `Strict-Transport-Security`.

## 22. Timeouts e negação de serviço — 🟡

**Feito:** `ReadHeaderTimeout` de 5 s, `ReadTimeout` de 15 s, `WriteTimeout` de 30 s, `IdleTimeout` de 60 s, timeout de 15 s por requisição no roteador, corpo limitado a 1 MB, paginação com máximo de 100 itens e rate limit (item 15).

No WebSocket: no máximo 20 conexões abertas por IP, mensagem do cliente limitada a 4 KB, 5 s para mandar o token, ping a cada 30 s e conexão fechada quando o token vence. O WebSocket público só abre para código válido e conta no rate limit do rastreio, e o `Origin` precisa estar na lista do CORS.

**Falta:** limite de conexões no pool do banco ajustado para produção.

## 23. Falsificação de IP — ✅

O `X-Forwarded-For` só é lido com `TRUST_PROXY=true`, que deve ser ligado apenas atrás do load balancer em produção. Fora disso, o IP é o da conexão. Mesmo com ele ligado, a API usa só o **último** IP do cabeçalho, que é o que o load balancer acrescenta; os da esquerda vêm do cliente e podem ser inventados. Por isso não usamos o `middleware.RealIP` do chi, que pega o primeiro. Testes em `internal/server` cobrem os dois casos.

## 24. Enumeração de e-mails — ✅

Quando o e-mail não existe, o login compara a senha com um hash bcrypt fixo, então o tempo é o mesmo dos dois jeitos (medido: ~80 a 90 ms em ambos). A mensagem também é a mesma, e o bloqueio por falhas vale para qualquer e-mail. O cadastro de motorista (409 "e-mail já usado") só é acessível à transportadora. O cadastro aberto de transportadora também responde 409 para e-mail usado, o que revela que o e-mail tem conta; é o mesmo trade-off de qualquer cadastro aberto, e a rota divide o rate limit do login (10/min por IP).

## 25. Banco de dados exposto — 🟡

**Desenvolvimento:** ✅ o Postgres do compose publica a porta só em `127.0.0.1`, então outras máquinas da rede não chegam nele.

**Produção**
- Banco em sub-rede privada, sem IP público, acessível só pela API.
- `sslmode=require` na conexão.
- Usuário da aplicação sem permissão de alterar schema; migrations com usuário próprio.
- Backups automáticos e criptografados.

## 26. Logs e auditoria — 🟡

**Feito:** logs estruturados em JSON com request ID; erro interno logado com detalhe; falha de login logada com IP e impressão do e-mail.

**Falta**
- Logar também 429, 403 e token inválido de forma pesquisável. Sem senha, token ou dado pessoal no log.
- Auditoria de negócio da edição de entregas (`PATCH`). Mudanças de status já ficam em `delivery_events`, com quem fez e quando.
- Alertas em produção para pico de falhas de login e de 5xx.

## 27. LGPD e retenção de dados — ✅

O sistema guarda nome, e-mail e endereço de destinatários.

**Feito**
- Rastreio público sem dados pessoais além do primeiro nome (item 16).
- O link público deixa de funcionar 30 dias depois de a entrega ser entregue ou da última falha (responde 404, como um código inexistente).
- E-mail e push de notificação levam só o primeiro nome, o código e o status, sem endereço nem observação do motorista. O nome passa pelo `html/template`, que escapa HTML.
- As inscrições de push são apagadas quando a entrega é entregue ou quando o serviço de push diz que expiraram.

- Retenção: `RETENTION_DAYS` (padrão 90, mínimo 31) dias depois de entregue ou da última falha, a API apaga nome, e-mail e endereço do destinatário e as observações dos eventos (texto livre que pode citar pessoas). A entrega e o histórico de status ficam para os relatórios, e a entrega anonimizada não aceita mais alteração (409), para os dados não voltarem. Roda a cada hora em `internal/retention`, testado com Postgres real.
- Coleta só o necessário: sem CPF, telefone etc. enquanto não houver uso.

**Falta**
- Backups seguem a mesma regra quando existirem (etapa 7).
