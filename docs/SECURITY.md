# Segurança

Cada risco, como o projeto se protege e em que pé está.

Legenda: ✅ feito · 🔜 planejado (com a etapa) · ➖ não se aplica hoje (com a regra para quando se aplicar)

## Itens

| # | Risco | Como o projeto se protege | Status |
| --- | --- | --- | --- |
| 1 | **Env exposta** | `.env` no `.gitignore` e no `.dockerignore`; `.env.example` só com valores falsos; `JWT_SECRET` com no mínimo 32 caracteres ou a API não sobe. No front, nada secreto em variáveis `VITE_*` (vão para o navegador). Em produção, segredos no AWS SSM Parameter Store. Gitleaks na CI barra segredo commitado. | ✅ local · 🔜 SSM (etapa 7), gitleaks (etapa 4) |
| 2 | **Validação no front-end** | Formulários validados com zod, mesmas regras da API, só para dar resposta rápida ao usuário. Nunca é a única barreira. | 🔜 etapa 3 |
| 3 | **Validação no back-end** | Toda entrada validada no service. JSON com campos desconhecidos é recusado; corpo limitado a 1 MB; IDs e paginação validados; erro 422 lista os campos. | ✅ |
| 4 | **SQL Injection** | Todo SQL é escrito em arquivos `.sql` e gerado pelo sqlc com parâmetros (`$1`, `$2`). Nenhuma concatenação de SQL. | ✅ |
| 5 | **Autenticação fraca** | Senha com no mínimo 8 caracteres; mesma mensagem para e-mail inexistente e senha errada (não revela quem tem conta). Melhorias: mínimo 10, recusar senhas comuns, e em produção a API não sobe com a senha padrão do admin. | ✅ base · 🔜 regras extras |
| 6 | **IDOR** | Rotas de admin exigem papel admin. Rotas do motorista filtram por `driver_id = usuário do token` e respondem 404 para entrega de outro motorista. Rastreio público usa código aleatório, nunca o ID sequencial. | ✅ admin · 🔜 motorista (etapa 2) |
| 7 | **Senhas direto no banco** | Senhas guardadas com bcrypt (custo 10, com salt). O hash nunca sai em respostas nem em logs. | ✅ |
| 8 | **Força bruta** | Limite de tentativas de login por IP e por e-mail, com bloqueio temporário que cresce a cada falha. bcrypt já deixa cada tentativa cara. | 🔜 próximo |
| 9 | **Bloquear durante envio** | Front desabilita o botão e mostra carregando enquanto envia. API aceita `Idempotency-Key` no `POST /deliveries`, para um clique duplo não criar duas entregas. | 🔜 etapa 3 |
| 10 | **CSRF** | Hoje o token vai no header `Authorization`, que o navegador não envia sozinho, então não há CSRF. Quando entrar o refresh token em cookie: `SameSite=Strict`, rota de refresh só por POST e checagem do header `Origin`. | ✅ hoje · 🔜 com o cookie |
| 11 | **Upload sem validação** | A v1 não tem upload. Se entrar foto de comprovante: upload direto no S3 por URL pré-assinada, tamanho máximo, tipo conferido pelos bytes do arquivo (não pela extensão), nome aleatório e bucket privado. | ➖ |
| 12 | **Revelando informação** | Erro inesperado vira `500 internal error` sem detalhes; o detalhe só vai para o log. Sem stack trace na resposta. Login não diz se o e-mail existe. Health check não mostra versões. | ✅ |
| 13 | **Dependências vulneráveis** | `govulncheck` e `npm audit` na CI, Dependabot abrindo PR de atualização, imagens base fixadas e atualizadas. | 🔜 etapa 4 |
| 14 | **Tokens mal otimizados** | JWT assinado com HS256, algoritmo conferido (recusa `none`), expiração obrigatória. Melhorias: access token de 15 minutos guardado só em memória no front, refresh token rotativo em cookie, salvo como hash no banco e revogável no logout. | ✅ base · 🔜 refresh token |
| 15 | **Rate limit** | Limite global por IP em toda a API e limite mais rígido em login e no rastreio público. Resposta 429 com `Retry-After`. | 🔜 próximo |
| 16 | **Dados sensíveis expostos** | Respostas usam structs próprias, nunca o modelo do banco. Página pública mostra só o primeiro nome do destinatário, o status e o histórico, sem e-mail nem endereço. Logs sem senha, token ou e-mail completo. HTTPS em produção. | ✅ API · 🔜 página pública (etapa 2) |
| 17 | **SSRF** | A API não faz requisições para URLs enviadas pelo usuário. Se um dia fizer (por exemplo webhooks): lista de domínios permitidos, bloqueio de IPs privados e de metadados da AWS (`169.254.169.254`), e timeout curto. | ➖ |
| 18 | **Cookies inseguros** | Quando houver cookie (refresh token): `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/auth` e expiração curta. | ➖ hoje · 🔜 com o refresh token |
| 19 | **CORS** | Lista de origens permitidas vinda de variável de ambiente (`CORS_ORIGINS`), sem `*`. Só os métodos e headers usados. | 🔜 próximo |

## Itens extras

| # | Risco | Como o projeto se protege | Status |
| --- | --- | --- | --- |
| 20 | **Cabeçalhos de segurança** | `Strict-Transport-Security`, `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` (contra clickjacking) e `Referrer-Policy`. | 🔜 próximo |
| 21 | **XSS** | React escapa o conteúdo por padrão; proibido `dangerouslySetInnerHTML`; CSP restritiva. | 🔜 etapa 3 |
| 22 | **Mass assignment** | O corpo é lido em structs só com os campos permitidos e campos extras são recusados. O status não muda pelo `PATCH`, só por evento. | ✅ |
| 23 | **Negação de serviço** | Timeout de leitura de headers e de 15 s por requisição, corpo limitado a 1 MB, paginação com máximo de 100. No WebSocket: limite de conexões por IP e de tamanho de mensagem. | ✅ HTTP · 🔜 WebSocket |
| 24 | **WebSocket** | Checagem do `Origin` no handshake, canal só de leitura e só para um código de rastreio. | 🔜 etapa 5 |
| 25 | **Enumeração de códigos** | Código de rastreio com 10 caracteres aleatórios (cerca de 50 bits) e rate limit no rastreio público. | ✅ código · 🔜 rate limit |
| 26 | **Menor privilégio** | Container roda como usuário sem root (distroless nonroot). Em produção: usuário do banco sem superuser, RDS em subnet privada, IAM só com o necessário. | ✅ container · 🔜 AWS |
| 27 | **Transporte** | HTTPS em tudo em produção; conexão com o banco com `sslmode=require`. | 🔜 etapa 7 |
| 28 | **Logs e auditoria** | Logs estruturados com ID da requisição; cada evento de status grava quem fez e quando; falhas de login registradas. | ✅ base · 🔜 auditoria |
| 29 | **Backups** | Snapshots automáticos do RDS. | 🔜 etapa 7 |

## Como reportar

Encontrou um problema? Abra uma issue sem detalhes sensíveis ou fale comigo pelo LinkedIn.
