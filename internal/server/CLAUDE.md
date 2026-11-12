# internal/server: rotas, middlewares e log

## Regras

- Ordem dos middlewares globais: request ID, IP do cliente, log, recover, headers de segurança, CORS, rate limit global (120/min por IP). Rotas HTTP com timeout de 15 s; WebSockets fora dele.
- IP do cliente: `X-Forwarded-For` só com `TRUST_PROXY=true`, e só o **último** valor. Não use o `middleware.RealIP` do chi, que pega o primeiro (falsificável).
- Rate limit e bloqueio contam a rede `/64` inteira no IPv6.
- Log próprio em JSON pelo `slog`, uma linha por requisição. Grava a rota (`/public/tracking/{code}`), nunca o caminho nem a query string, e nunca senha, token ou corpo. 401, 403 e 429 saem como aviso com o motivo; os middlewares de autenticação acrescentam quem fez por `httpx.AddLogAttrs`.
- Rota inexistente (404) e método errado (405) respondem no mesmo JSON dos outros erros; cliente que desistiu recebe 499 e consulta lenta, 504.

## Ver também

Decisões [002](../../docs/decisoes/002-ip-do-cliente-atras-de-proxy-ultimo.md), [017](../../docs/decisoes/017-trust-proxy-liga-desliga-a-leitura-de.md), [093](../../docs/decisoes/093-log-de-requisicao-proprio-em-json-pelo.md). Segurança: itens 15, 21, 22, 23 e 26.
