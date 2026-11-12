# 024. `Idempotency-Key` guarda o id da entrega, não a resposta

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

O reenvio devolve a entrega como está agora; mais simples que guardar o corpo, e a chave é reservada na mesma transação da criação, então pedidos simultâneos criam uma entrega só.

## Decisão

`Idempotency-Key` guarda o id da entrega, não a resposta.

Alternativas descartadas: guardar status + corpo da resposta.

## Consequência

Não registrada além do contexto.
