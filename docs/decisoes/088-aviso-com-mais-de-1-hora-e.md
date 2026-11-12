# 088. Aviso com mais de 1 hora é descartado também no worker

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

Com o worker parado a fila acumula, e mandar "Saiu para entrega" horas depois confunde mais do que ajuda.

## Decisão

Aviso com mais de 1 hora é descartado também no worker.

## Consequência

Não registrada além do contexto.
