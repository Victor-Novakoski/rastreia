# 089. Sem as chaves VAPID, o worker esvazia a fila de push

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

A exchange manda todo evento para as duas filas; sem quem consumisse, a fila de push crescia para sempre.

## Decisão

Sem as chaves VAPID, o worker esvazia a fila de push.

## Consequência

Não registrada além do contexto.
