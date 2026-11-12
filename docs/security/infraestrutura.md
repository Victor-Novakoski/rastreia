# Segurança: infraestrutura e deploy

Itens 1, 13, 15, 17, 22, 23 e 25: segredos, dependências, rate limit, SSRF, timeouts, IP do cliente e banco. Resumo e legenda em [SECURITY.md](../SECURITY.md).

## 1. Variáveis de ambiente expostas — 🟡

**Feito**
- `.env` está no `.gitignore` e no `.dockerignore` (não entra no git nem na imagem).
- Segredos só por variável de ambiente; `.env.example` com valores de desenvolvimento.
- A API não sobe com `JWT_SECRET` com menos de 32 caracteres.
- Com `APP_ENV=production`, a API **recusa subir** com o `JWT_SECRET` ou o `ADMIN_PASSWORD` de desenvolvimento, ou com origem de CORS sem `https` (`internal/config`, com testes).
- gitleaks na CI varre todo o histórico do git a cada PR, procurando chave ou senha commitada por engano.

**Falta**
- Em produção, segredos no gerenciador de segredos ou nas variáveis de ambiente do provedor, nunca em arquivo ([decisão 097](../decisoes/097-deploy-sem-custo-de-hospedagem.md), issue [#44](https://github.com/Victor-Novakoski/rastreia/issues/44)).

## 13. Dependências vulneráveis — ✅

**Feito**
- `govulncheck` rodado em 01/10/2026: **nenhuma vulnerabilidade alcançável pelo código**. Ele aponta o GO-2026-5932, no pacote `openpgp` de `golang.org/x/crypto`, que o projeto não usa (só usamos `bcrypt`).
- Na etapa 2 o testcontainers trouxe o `moby/go-archive` v0.2.0 com o GO-2026-6253 (alcançável só pelo código de teste); atualizado para a v0.3.0, que corrige.
- Imagem final distroless, sem shell nem gerenciador de pacotes, rodando como usuário não-root.
- `govulncheck` roda na CI em todo PR.
- `npm audit --omit=dev --audit-level=high` roda na CI do front.
- Dependabot abre PR semanal para a `develop` com atualizações de módulos Go, pacotes npm do front, imagens Docker e GitHub Actions.
- As GitHub Actions são fixadas pelo hash do commit, não pela tag, porque tags podem ser trocadas por quem invadir o repositório da action.
- Trivy varre a imagem final na CI e falha o PR em vulnerabilidade HIGH ou CRITICAL que já tenha correção.

## 15. Rate limit — ✅

**Feito**
- Limite global de 120 requisições por minuto por IP e de 10 por minuto em cada rota de sessão (`/auth/login`, `/auth/signup`, `/auth/refresh` e `/auth/logout`, cada uma com o seu contador), com 429 e `Retry-After` (`internal/server/middleware.go`, com testes).
- Limite próprio de 30 por minuto por IP no rastreio público, dividido entre a página (`/public/tracking/{code}`), o WebSocket dela e a inscrição de push, para dificultar a varredura de códigos.
- O IP usado é o da conexão, a não ser que `TRUST_PROXY=true` (item 23). No IPv6 conta a rede `/64` inteira, porque cada aparelho costuma receber uma `/64` e trocar de endereço dentro dela.
- A página de rastreio não chama `/auth/refresh`: quem só acompanha uma entrega não gasta as renovações do IP, que numa rede compartilhada são as mesmas de quem está logado.
- Com `REDIS_URL`, os contadores ficam no Redis e valem para todas as instâncias; se o Redis cair, cada instância volta a contar em memória até ele voltar.
- Em produção o Redis não deve ficar exposto na internet; no compose de desenvolvimento ele só escuta em `127.0.0.1`.

## 17. SSRF — ✅

O único lugar que chama uma URL vinda de fora é o Web Push: o navegador entrega o `endpoint` da inscrição e o worker faz um POST nele. Feito:
- O endpoint só é aceito com `https`, sem porta nem usuário, e com o host numa lista fechada dos serviços de push dos navegadores (Google, Mozilla, Microsoft, Apple). Qualquer outro host, IP ou `localhost` responde 422 (`internal/push`, testado).
- O worker não segue redirecionamento e tem timeout de 15s.
- A chave `p256dh` precisa ser um ponto válido da curva P-256, e há no máximo 10 navegadores por entrega.

Se outra URL vinda do usuário aparecer (webhook, geocodificação, imagem por URL):
- Lista de hosts permitidos.
- Bloquear IPs privados, loopback e o endereço de metadados da nuvem (`169.254.169.254`), verificando o IP **depois** de resolver o DNS.
- Timeout curto e sem seguir redirecionamentos.
- Se o provedor tiver serviço de metadados, exigir acesso autenticado (como o IMDSv2 da AWS).

## 22. Timeouts e negação de serviço — 🟡

**Feito:** `ReadHeaderTimeout` de 5 s, `ReadTimeout` de 15 s, `WriteTimeout` de 30 s, `IdleTimeout` de 60 s, timeout de 15 s por requisição no roteador, corpo limitado a 1 MB, paginação com máximo de 100 itens e rate limit (item 15).

No WebSocket: no máximo 20 conexões abertas por IP (ou por rede `/64` no IPv6) em cada instância da API, mensagem do cliente limitada a 4 KB, 5 s para mandar o token, ping a cada 30 s e conexão fechada quando o token vence. O WebSocket público só abre para código válido e conta no rate limit do rastreio, e, quando o navegador manda `Origin`, ele precisa estar na lista do CORS.

**Falta:** limite de conexões no pool do banco ajustado para produção (issue [#51](https://github.com/Victor-Novakoski/rastreia/issues/51)).

## 23. Falsificação de IP — ✅

O `X-Forwarded-For` só é lido com `TRUST_PROXY=true`, que deve ser ligado apenas atrás do load balancer em produção. Fora disso, o IP é o da conexão. Mesmo com ele ligado, a API usa só o **último** IP do cabeçalho, que é o que o load balancer acrescenta; os da esquerda vêm do cliente e podem ser inventados. Por isso não usamos o `middleware.RealIP` do chi, que pega o primeiro. Testes em `internal/server` cobrem os dois casos.

## 25. Banco de dados exposto — 🟡

**Desenvolvimento:** ✅ o Postgres do compose publica a porta só em `127.0.0.1`, então outras máquinas da rede não chegam nele.

**Produção**
- Banco em sub-rede privada, sem IP público, acessível só pela API.
- `sslmode=require` na conexão.
- Usuário da aplicação sem permissão de alterar schema; migrations com usuário próprio.
- Backups automáticos e criptografados.
