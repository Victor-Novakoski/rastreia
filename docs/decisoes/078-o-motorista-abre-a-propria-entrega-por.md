# 078. O motorista abre a própria entrega por `GET /deliveries/{id}`

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

O app achava a entrega entre as 100 da lista e uma entrega antiga em aberto sumia para quem entrega muito; a regra de dono (`visible`) já existia no service e responde 404 para as dos outros.

## Decisão

O motorista abre a própria entrega por `GET /deliveries/{id}`.

Alternativas descartadas: rota nova `/me/deliveries/{id}`, que repetiria o handler.

## Consequência

Não registrada além do contexto.
