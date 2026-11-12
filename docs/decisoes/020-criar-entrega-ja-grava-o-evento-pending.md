# 020. Criar entrega já grava o evento `pending`

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

O histórico começa na criação, com quem criou; entregas antigas ganharam um evento na migration (com `created_by` nulo, de sistema).

## Decisão

Criar entrega já grava o evento `pending`.

## Consequência

Não registrada além do contexto.
