# internal/notify (e internal/push): e-mail e Web Push

Fluxo completo no diagrama de [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md#notificações).

## Regras

- `delivery_events` é o outbox: o relay da API lê com `FOR UPDATE SKIP LOCKED`, publica com confirmação do RabbitMQ e só então marca `published_at`.
- Pelo menos uma vez, sem deduplicar no worker.
- Cada canal tem fila própria com `.retry` (TTL de 30 s) e `.dead` (depois de 5 tentativas, contadas no `x-death`; mensagem inválida vai direto).
- Aviso com mais de 1 hora é descartado no relay e no worker.
- O e-mail não lê o banco (a mensagem leva nome e e-mail). O push lê `push_subscriptions`, por isso o worker recebe `DATABASE_URL` quando tem as chaves VAPID. Sem as chaves, o worker esvazia a fila de push.
- Endpoint de push (`internal/push`) só `https`, sem porta nem usuário, com host na lista fechada dos serviços dos navegadores: é a defesa contra SSRF. No máximo 10 navegadores por entrega; inscrições somem quando a entrega é entregue ou o serviço responde 404/410.
- Notificação leva só primeiro nome, código e status.
- Desenvolvimento: e-mails no Mailpit (`:8025`). `worker vapid` gera as chaves.

## Ver também

Decisões [048](../../docs/decisoes/048-delivery-events-como-outbox-com-um-relay.md) a [057](../../docs/decisoes/057-service-worker-registrado-so-ao-ativar-os.md), [088](../../docs/decisoes/088-aviso-com-mais-de-1-hora-e.md), [089](../../docs/decisoes/089-sem-as-chaves-vapid-o-worker-esvazia.md). Segurança: item 17.
