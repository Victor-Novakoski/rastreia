# Segurança

Como o Rastreia trata cada risco, o que já está feito e o que falta. As tarefas pendentes estão em [TASKS.md](TASKS.md) e o checklist rápido para cada mudança está em [RULES.md](RULES.md#3-segurança).

Situação revisada em 01/10/2026, depois da etapa 2 (eventos e rastreio público).

**Legenda:** ✅ feito · 🟡 parcial · 🔴 pendente · ⚪ ainda não se aplica (regra definida para quando se aplicar)

## Resumo

| # | Risco | Situação | Prioridade |
| --- | --- | --- | --- |
| 1 | Variáveis de ambiente expostas | 🟡 | Etapas 4 e 7 |
| 2 | Validação no front-end | ⚪ | — |
| 3 | Validação no back-end | ✅ | — |
| 4 | SQL Injection | ✅ | — |
| 5 | Autenticação fraca | 🟡 | Baixa |
| 6 | IDOR | ✅ | — |
| 7 | Senhas no banco | ✅ | — |
| 8 | Força bruta | ✅ | — |
| 9 | Envio duplicado | 🟡 | Etapa 3 (front) |
| 10 | CSRF | ⚪ | — |
| 11 | Upload sem validação | ⚪ | — |
| 12 | Vazamento de informação | 🟡 | Média |
| 13 | Dependências vulneráveis | 🟡 | Média |
| 14 | Tokens | 🟡 | Média |
| 15 | Rate limit | 🟡 | Etapa 5 (Redis) |
| 16 | Dados sensíveis expostos | 🟡 | Etapa 7 (HTTPS) |
| 17 | SSRF | ⚪ | — |
| 18 | Cookies inseguros | ⚪ | — |
| 19 | CORS | ✅ | — |
| 20 | XSS | ⚪ | Etapa 3 |
| 21 | Headers de segurança | ✅ | — |
| 22 | Timeouts e negação de serviço | 🟡 | Etapa 5 |
| 23 | Falsificação de IP | ✅ | — |
| 24 | Enumeração de e-mails | ✅ | — |
| 25 | Banco de dados exposto | 🟡 | Etapa 7 |
| 26 | Logs e auditoria | 🟡 | Média |
| 27 | LGPD e retenção de dados | 🟡 | Etapa 6 |

Os itens 1 a 19 são a lista original; os itens 20 a 27 completam a cobertura.

---

## 1. Variáveis de ambiente expostas — 🟡

**Feito**
- `.env` está no `.gitignore` e no `.dockerignore` (não entra no git nem na imagem).
- Segredos só por variável de ambiente; `.env.example` com valores de desenvolvimento.
- A API não sobe com `JWT_SECRET` com menos de 32 caracteres.
- Com `APP_ENV=production`, a API **recusa subir** com o `JWT_SECRET` ou o `ADMIN_PASSWORD` de desenvolvimento, ou com origem de CORS sem `https` (`internal/config`, com testes).

**Falta**
- Varredura de segredos no CI (gitleaks) para pegar chave commitada por engano.
- Em produção (AWS), segredos no Secrets Manager ou SSM Parameter Store, nunca em arquivo.

## 2. Validação no front-end — ⚪

Entra na etapa 3. Regra: o front valida para dar **feedback rápido** ao usuário (campos obrigatórios, formato de e-mail, tamanho), mas nunca é a proteção. Toda regra do front existe também no back. As mensagens de `fields` do 422 aparecem embaixo do campo.

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

- Rotas de admin (`/drivers`, `/deliveries`, `/deliveries/{id}`) respondem 403 para motorista.
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
- O bloqueio fica em memória; quando houver mais de uma instância da API, passa para o Redis.

## 9. Bloquear durante envio (envio duplicado) — 🟡

**Back (feito)**
- `POST /deliveries` aceita o cabeçalho `Idempotency-Key`: a mesma chave do mesmo usuário em até 24h devolve a entrega criada na primeira vez (com `Idempotent-Replayed: true`) em vez de criar outra. A mesma chave com outro corpo responde 422.
- A chave é reservada na mesma transação que cria a entrega, então duas requisições simultâneas com a mesma chave criam uma entrega só (teste de integração com 10 requisições em paralelo).
- Eventos de status: transição repetida (ex.: `picked_up` → `picked_up`) responde 409, então reenviar não duplica histórico. Dois eventos simultâneos na mesma entrega: só um é aplicado, o outro recebe 409 (a troca de status confere o status anterior no `UPDATE`).

**Front (etapa 3):** botão desabilitado e com indicador de carregamento enquanto a requisição não volta, e uma `Idempotency-Key` gerada por formulário.

## 10. CSRF — ⚪

**Hoje não se aplica:** a autenticação é por cabeçalho `Authorization: Bearer`, que o navegador não envia sozinho, e a API não usa cookies.

**Se o refresh token for para cookie (item 14):** `SameSite=Strict`, cookie restrito ao caminho `/auth/refresh` e verificação do cabeçalho `Origin` contra a lista de origens permitidas. Se algum dia a sessão inteira for por cookie, token anti-CSRF obrigatório nas rotas que alteram dados.

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

## 13. Dependências vulneráveis — 🟡

**Feito**
- `govulncheck` rodado em 01/10/2026: **nenhuma vulnerabilidade alcançável pelo código**. Ele aponta o GO-2026-5932, no pacote `openpgp` de `golang.org/x/crypto`, que o projeto não usa (só usamos `bcrypt`).
- Na etapa 2 o testcontainers trouxe o `moby/go-archive` v0.2.0 com o GO-2026-6253 (alcançável só pelo código de teste); atualizado para a v0.3.0, que corrige.
- Imagem final distroless, sem shell nem gerenciador de pacotes, rodando como usuário não-root.

**Falta**
- `govulncheck` no CI bloqueando merge.
- Dependabot (ou Renovate) para módulos Go, imagens Docker e GitHub Actions.
- Varredura da imagem Docker (Trivy) no CI.

## 14. Tokens mal otimizados — 🟡

**Feito:** algoritmo fixo, expiração, claims mínimos (`sub`, `role`, `iat`, `exp`), segredo com tamanho mínimo.

**Problemas**
- Token de 24h sem como revogar: motorista desligado continua com acesso por até 24h.
- Sem `iss`/`aud`.

**Plano**
- Access token curto (15 min) no cabeçalho, guardado só em memória no front.
- Refresh token opaco e aleatório, em cookie `HttpOnly`, salvo **com hash** no banco, trocado a cada uso (rotação) e revogável (logout, troca de senha, desativação do usuário).
- Reuso de refresh token já trocado revoga toda a sessão (sinal de roubo).
- Adicionar `iss` e `aud` e validá-los.

## 15. Rate limit — 🟡

**Feito**
- Limite global de 120 requisições por minuto por IP e de 10 por minuto no login, com 429 e `Retry-After` (`internal/server/middleware.go`, com testes).
- Limite próprio de 30 por minuto por IP no rastreio público (`/public/tracking/{code}`), para dificultar a varredura de códigos.
- O IP usado é o da conexão, a não ser que `TRUST_PROXY=true` (item 23).

**Falta**
- Mover os contadores para o Redis quando houver mais de uma instância.

## 16. Dados sensíveis expostos — 🟡

**Feito**
- Hash de senha nunca sai da API; `.env` fora do git; logs não registram corpo de requisição nem token.
- O rastreio público tem resposta própria (`delivery.Tracking`): código, status, histórico (status e horário) e primeiro nome do destinatário. **Não** inclui e-mail, endereço, sobrenome, motorista, ids internos nem as observações dos eventos (texto livre do motorista pode ter dado pessoal). Testado no serviço e pela API.

**Falta**
- HTTPS obrigatório em produção (HSTS).
- Banco exposto (item 25).

## 17. SSRF — ⚪

A API não faz requisições para URLs informadas pelo usuário. Se passar a fazer (webhook, geocodificação, imagem por URL):
- Lista de hosts permitidos.
- Bloquear IPs privados, loopback e o endereço de metadados da nuvem (`169.254.169.254`), verificando o IP **depois** de resolver o DNS.
- Timeout curto e sem seguir redirecionamentos.
- Na AWS, IMDSv2 obrigatório.

## 18. Cookies inseguros — ⚪

Não há cookies hoje. Qualquer cookie que for criado: `HttpOnly`, `Secure`, `SameSite=Strict` (ou `Lax` com justificativa), `Path` restrito, prefixo `__Host-` quando possível e validade curta.

## 19. CORS — ✅

- Lista exata de origens vinda de `CORS_ORIGINS` (padrão: `http://localhost:5173`, o Vite), sem `*`. Em produção só aceita `https`.
- Só os métodos (`GET`, `POST`, `PATCH`) e cabeçalhos (`Authorization`, `Content-Type`) usados.
- Sem `credentials` por enquanto; entra só se o refresh por cookie exigir (item 18).

## 20. XSS — ⚪

Etapa 3. React já escapa o conteúdo; proibido `dangerouslySetInnerHTML` com dado vindo da API. Content-Security-Policy restritiva no front. O token de acesso fica em memória, não em `localStorage`.

## 21. Headers de segurança — ✅

Middleware em todas as respostas: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` e `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'` (contra clickjacking), `Referrer-Policy: no-referrer`, `Cache-Control: no-store` e, em produção, `Strict-Transport-Security`.

## 22. Timeouts e negação de serviço — 🟡

**Feito:** `ReadHeaderTimeout` de 5 s, `ReadTimeout` de 15 s, `WriteTimeout` de 30 s, `IdleTimeout` de 60 s, timeout de 15 s por requisição no roteador, corpo limitado a 1 MB, paginação com máximo de 100 itens e rate limit (item 15).

**Falta:** limite de conexões no pool do banco ajustado para produção, e no WebSocket (etapa 5) um limite de conexões por IP e de tamanho de mensagem.

## 23. Falsificação de IP — ✅

O `X-Forwarded-For` só é lido com `TRUST_PROXY=true`, que deve ser ligado apenas atrás do load balancer em produção. Fora disso, o IP é o da conexão. Teste em `internal/server` garante que um `X-Forwarded-For` falso não escapa do rate limit.

## 24. Enumeração de e-mails — ✅

Quando o e-mail não existe, o login compara a senha com um hash bcrypt fixo, então o tempo é o mesmo dos dois jeitos (medido: ~80 a 90 ms em ambos). A mensagem também é a mesma, e o bloqueio por falhas vale para qualquer e-mail. O cadastro de motorista (409 "e-mail já usado") só é acessível ao admin.

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

## 27. LGPD e retenção de dados — 🟡

O sistema guarda nome, e-mail e endereço de destinatários.

**Feito**
- Rastreio público sem dados pessoais além do primeiro nome (item 16).
- O link público deixa de funcionar 30 dias depois de a entrega ser entregue ou da última falha (responde 404, como um código inexistente).

**Falta**
- Definir por quanto tempo dados pessoais de entregas concluídas ficam guardados no banco e anonimizar depois (etapa 6).
- Coletar só o necessário (sem CPF, telefone etc. enquanto não houver uso).
