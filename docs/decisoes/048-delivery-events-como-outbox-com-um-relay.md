# 048. `delivery_events` como outbox, com um relay na API que publica no RabbitMQ

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Publicar direto depois do commit perde a notificação se a API cair ou o RabbitMQ estiver fora; com o outbox o evento e a notificação são gravados juntos. `SKIP LOCKED` deixa várias instâncias rodarem o relay.

## Decisão

`delivery_events` como outbox, com um relay na API que publica no RabbitMQ.

Alternativas descartadas: publicar após o commit; tabela `outbox` separada, que duplicaria o evento.

## Consequência

Não registrada além do contexto.
