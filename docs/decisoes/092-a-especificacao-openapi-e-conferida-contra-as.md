# 092. A especificação OpenAPI é conferida contra as rotas num teste (`TestOpenAPI`)

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

O arquivo tinha YAML inválido e respostas faltando sem ninguém perceber; o teste falha se uma rota servida não estiver documentada, ou o contrário.

## Decisão

A especificação OpenAPI é conferida contra as rotas num teste (`TestOpenAPI`).

Alternativas descartadas: lint da especificação na CI, mais uma ferramenta para manter.

## Consequência

Não registrada além do contexto.
