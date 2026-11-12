# 017. `TRUST_PROXY` liga/desliga a leitura de `X-Forwarded-For`

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

Mais simples que uma lista de proxies; em produção só há o load balancer na frente.

## Decisão

`TRUST_PROXY` liga/desliga a leitura de `X-Forwarded-For`.

Alternativas descartadas: `TRUSTED_PROXIES` com faixas de IP.

## Consequência

Não registrada além do contexto.
