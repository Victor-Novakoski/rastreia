# 041. `carrier_id` no token (`cid`) e filtro no service, não em middleware

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O token já diz quem é e de onde; os services recebem o `auth.Claims` e cada query filtra ou confere a transportadora, o que os testes de IDOR cobrem.

## Decisão

`carrier_id` no token (`cid`) e filtro no service, não em middleware.

Alternativas descartadas: buscar a transportadora no banco a cada requisição.

## Consequência

Não registrada além do contexto.
