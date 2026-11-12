# 081. Trava entre abas (`navigator.locks`) na renovação do token

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

Abas abertas juntas usavam o mesmo refresh token, e a API, que trata reuso como roubo, derrubava a sessão. A trava resolve no navegador sem afrouxar a regra na API.

## Decisão

Trava entre abas (`navigator.locks`) na renovação do token.

Alternativas descartadas: período de tolerância na API, que deixaria um token roubado valer por alguns segundos.

## Consequência

Não registrada além do contexto.
