# 061. Uma inscrição por padrão (`PSUBSCRIBE rastreia:live:*`) por instância, que repassa para o broker em memória

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Uma conexão no Redis por instância, não uma por navegador.

## Decisão

Uma inscrição por padrão (`PSUBSCRIBE rastreia:live:*`) por instância, que repassa para o broker em memória.

Alternativas descartadas: um `SUBSCRIBE` por WebSocket.

## Consequência

Não registrada além do contexto.
