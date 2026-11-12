# 049. Retry com fila de espera (TTL de 30s) e fila de falhas depois de 5 tentativas

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O RabbitMQ conta as tentativas no header `x-death`, sem estado no worker; mensagem inválida vai direto para a fila de falhas.

## Decisão

Retry com fila de espera (TTL de 30s) e fila de falhas depois de 5 tentativas.

Alternativas descartadas: requeue imediato, que vira loop; plugin de delayed message.

## Consequência

Não registrada além do contexto.
