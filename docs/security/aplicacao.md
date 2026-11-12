# Segurança: aplicação e navegador

Itens 2, 3, 4, 6, 9, 11, 19, 20 e 21: validação, SQL injection, IDOR, envio duplicado, upload, CORS, XSS e headers. Resumo e legenda em [SECURITY.md](../SECURITY.md).

## 2. Validação no front-end — ✅

Feito no painel da transportadora e no app do motorista: campos com `required`, `type="email"` e `maxLength` iguais aos limites da API, e os `fields` do 422 traduzidos e mostrados embaixo de cada campo (`web/src/lib/fields.ts`). Regra: o front valida para dar **feedback rápido** ao usuário (campos obrigatórios, formato de e-mail, tamanho), mas nunca é a proteção. Toda regra do front existe também no back.

## 3. Validação no back-end — ✅

- Validação centralizada no service com `apperr.Validator`, retornando 422 com todos os campos inválidos.
- Normalização (trim, e-mail em minúsculas) antes de validar.
- `httpx.Decode` limita o corpo a 1 MB e rejeita campos desconhecidos (impede *mass assignment*, ex.: mandar `"status"` ou `"role"` no corpo).
- Tamanho máximo nos textos, contado em caracteres (não em bytes): nome 120, e-mail 254, rua 200, número 20, complemento, bairro e cidade 100, ponto de referência 300, busca 100 e observação do evento 500. CEP com 8 dígitos, telefone com 10 ou 11 (com DDD) e UF da lista dos estados.
- Texto com o caractere nulo (`\u0000`) responde 400: o PostgreSQL não guarda esse caractere, e a requisição viraria 500.
- Senha com mais de 72 bytes (limite do bcrypt) volta 422 em vez de 500.
- O banco reforça com `CHECK` (status, papel e latitude/longitude juntas), `NOT NULL` e `UNIQUE`.

## 4. SQL Injection — ✅

Todo SQL fica em `internal/database/queries/*.sql` e o sqlc gera código com parâmetros (`$1`, `$2`). Não existe SQL montado com concatenação. Regra em [internal/CLAUDE.md](../../internal/CLAUDE.md): SQL só pelo sqlc. Filtros dinâmicos (ex.: status opcional) são resolvidos com `sqlc.narg`, não com string.

## 6. IDOR (acesso a recurso de outra pessoa pelo id) — ✅

- Cada transportadora é um tenant. O `carrier_id` vem do token, e toda leitura ou escrita confere a transportadora: listagens filtram na query SQL, buscas por id respondem **404** quando a entrega é de outra transportadora, e motorista de outra transportadora não pode ser atribuído (422). Testes: `TestCarrierIsolation` (service) e `TestIntegration_CarriersAreIsolated` (API com Postgres real).
- Rotas só da transportadora (`/drivers`, a lista e a criação em `/deliveries`, o `PATCH /deliveries/{id}` e `/summary`) respondem 403 para motorista.
- O motorista lista entregas por `/me/deliveries`, e o filtro `driver_id = <id do token>` fica **na query SQL** (`ListDriverDeliveries`): entrega de outro motorista nunca sai do banco.
- `GET /deliveries/{id}` e `GET`/`POST /deliveries/{id}/events` conferem se a entrega pertence ao motorista; se não pertencer (ou não tiver motorista), respondem **404**, igual a uma entrega inexistente, para não confirmar que o id existe.
- Rota do motorista: o pacote bipado é buscado pelo código e conferido com a transportadora do token; de outra transportadora responde 404, como inexistente, e de outro motorista, 409. Remover e reordenar só mexem na rota do próprio motorista. Testes: `TestPG_ScanRefused` e `TestIntegration_Route`.
- O rastreio público usa só o código aleatório (10 caracteres de um alfabeto de 32, cerca de 50 bits), nunca o id sequencial.
- Testes: unitários (`TestEvents_DriverOnlySeesOwnDeliveries`) e de integração pela API com Postgres real (`TestIntegration_DriversOnlyReachTheirOwnDeliveries`): motorista B tentando ler, alterar e listar entrega do motorista A.

## 9. Bloquear durante envio (envio duplicado) — ✅

**Back (feito)**
- `POST /deliveries` aceita o cabeçalho `Idempotency-Key`: a mesma chave do mesmo usuário em até 24h devolve a entrega criada na primeira vez (com `Idempotent-Replayed: true`) em vez de criar outra. A mesma chave com outro corpo responde 422.
- A chave é reservada na mesma transação que cria a entrega, então duas requisições simultâneas com a mesma chave criam uma entrega só (teste de integração com 10 requisições em paralelo).
- Eventos de status: transição repetida (ex.: `picked_up` → `picked_up`) responde 409, então reenviar não duplica histórico. Dois eventos simultâneos na mesma entrega: só um é aplicado, o outro recebe 409 (a troca de status confere o status anterior no `UPDATE`).

**Front (feito):** o `Button` fica desabilitado e com indicador enquanto a requisição não volta, e o formulário ignora um segundo envio. O formulário de nova entrega gera uma `Idempotency-Key` (`crypto.randomUUID()`), mantida se a rede falhar e trocada quando a API responde com erro.

## 11. Upload sem validação — ⚪

Não há upload hoje. Se o comprovante de entrega com foto entrar (hoje fora da v1, decisão no [PRD](../PRD.md)):
- Tamanho máximo (ex.: 5 MB) com `http.MaxBytesReader`.
- Tipo verificado pelo conteúdo do arquivo (*magic bytes*), não pela extensão nem pelo `Content-Type` enviado; só JPEG, PNG e WebP.
- Reprocessar a imagem no servidor para remover metadados (EXIF tem GPS).
- Nome gerado pelo servidor; arquivo em bucket privado, entregue por URL assinada e temporária.
- Nunca servir o arquivo pelo mesmo domínio da API como HTML.

## 19. CORS — ✅

- Lista exata de origens vinda de `CORS_ORIGINS` (padrão: `http://localhost:5173`, o Vite), sem `*`. Em produção só aceita `https`.
- Só os métodos (`GET`, `POST`, `PATCH`, `PUT`, `DELETE`) e cabeçalhos (`Authorization`, `Content-Type`, `Idempotency-Key`) usados.
- `credentials` liberado, porque o refresh token vai no cookie (item 18). Por isso a lista de origens nunca pode ter `*`.

## 20. XSS — ✅

- React escapa todo texto vindo da API. O oxlint barra `dangerouslySetInnerHTML` (`react/no-danger`), links `javascript:` e `target="_blank"` sem `rel="noopener noreferrer"`.
- O build injeta uma Content-Security-Policy no `index.html` (`web/csp.ts`): scripts e estilos só do próprio site, nada inline, e conexões só com a origem de `VITE_API_URL` (HTTP e WebSocket) e com o ViaCEP e o Nominatim (endereço da entrega). Imagens externas só os mapas do OpenStreetMap. No `npm run dev` a CSP fica de fora, porque o hot reload do Vite usa script inline.
- Para o ViaCEP e o Nominatim só vão CEP, rua, número, cidade e UF, nunca o nome, o telefone ou o e-mail do destinatário. O Nominatim é chamado no máximo uma vez por segundo, como pede a política de uso dele.
- O access token fica em memória, não em `localStorage`; o refresh token está num cookie `HttpOnly`. Um XSS não leva a sessão embora.

**Deploy (issue [#45](https://github.com/Victor-Novakoski/rastreia/issues/45)):** `frame-ancestors` não funciona em `<meta>`, então o host do front precisa mandar `Content-Security-Policy: frame-ancestors 'none'` (ou `X-Frame-Options: DENY`) no cabeçalho.

## 21. Headers de segurança — ✅

Middleware em todas as respostas: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` e `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'` (contra clickjacking), `Referrer-Policy: no-referrer`, `Cache-Control: no-store` e, em produção, `Strict-Transport-Security`.
