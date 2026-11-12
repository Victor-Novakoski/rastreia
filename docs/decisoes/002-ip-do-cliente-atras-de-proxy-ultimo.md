# 002. IP do cliente atrás de proxy = último valor do `X-Forwarded-For`

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

O `middleware.RealIP` do chi usa o primeiro, que o cliente pode falsificar (achado pelo golangci-lint).

## Decisão

IP do cliente atrás de proxy = último valor do `X-Forwarded-For`.

Alternativas descartadas: chi RealIP, httprate KeyByRealIP.

## Consequência

Não registrada além do contexto.
