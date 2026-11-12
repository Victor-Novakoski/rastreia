# 028. Endereço em colunas (CEP, rua, número...) e `address` montado pela API

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O CEP preenche o resto, a rota agrupa pelo endereço e a etiqueta sai certa; `address` continua como a linha inteira para quem só quer ler, e entregas antigas ficam só com ela.

## Decisão

Endereço em colunas (CEP, rua, número...) e `address` montado pela API.

Alternativas descartadas: JSON numa coluna, que o sqlc e as queries tratam pior; tabela de endereços, sem reuso que justifique.

## Consequência

Não registrada além do contexto.
