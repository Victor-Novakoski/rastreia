# 087. Entrega parada há mais de 1 ano também é anonimizada, e a anonimizada não muda mais

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

Uma entrega que nunca termina guardaria os dados para sempre; e uma edição que chegasse junto com a retenção gravaria os dados de volta, então as queries exigem `anonymized_at IS NULL` e a API responde 409.

## Decisão

Entrega parada há mais de 1 ano também é anonimizada, e a anonimizada não muda mais.

## Consequência

Não registrada além do contexto.
