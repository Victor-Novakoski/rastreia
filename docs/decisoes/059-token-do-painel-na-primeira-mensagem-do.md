# 059. Token do painel na primeira mensagem do WebSocket, não na URL

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O navegador não manda `Authorization` no WebSocket, e token na URL vai parar em log. A API fecha com o código 4001 quando o token vence, e o front renova e reconecta.

## Decisão

Token do painel na primeira mensagem do WebSocket, não na URL.

Alternativas descartadas: cookie, que exigiria mais checagem de CSRF; subprotocolo.

## Consequência

Não registrada além do contexto.
