# 060. Redis opcional (`REDIS_URL`): com ele, tempo real, rate limit e bloqueio de login valem para todas as instâncias; sem ele, ficam em memória

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Rodar a API sozinha (`make run`, testes) continua simples, e o compose sobe o Redis.

## Decisão

Redis opcional (`REDIS_URL`): com ele, tempo real, rate limit e bloqueio de login valem para todas as instâncias; sem ele, ficam em memória.

Alternativas descartadas: Redis obrigatório.

## Consequência

Não registrada além do contexto.
