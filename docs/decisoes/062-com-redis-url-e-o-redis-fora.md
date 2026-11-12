# 062. Com `REDIS_URL` e o Redis fora do ar, o login responde 503; o rate limit cai para memória

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Sem contador não há proteção contra força bruta, então o login espera; o rate limit em memória ainda protege cada instância.

## Decisão

Com `REDIS_URL` e o Redis fora do ar, o login responde 503; o rate limit cai para memória.

Alternativas descartadas: deixar logar sem contador.

## Consequência

Não registrada além do contexto.
