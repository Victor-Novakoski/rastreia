# internal/realtime: WebSocket e pub/sub

## Regras

- O servidor só envia. O rastreio público recebe o mesmo corpo do `GET`; o painel recebe só `{delivery_id, status}`, sem dado pessoal, e o front invalida o cache.
- O token do painel vai na primeira mensagem (5 s para chegar), nunca na URL. Token vencido fecha com o código 4001.
- Tópico do painel por transportadora (`deliveries:<id>`), escolhido pelo token.
- Com `REDIS_URL`, uma inscrição por padrão (`PSUBSCRIBE rastreia:live:*`) por instância, repassada ao broker em memória. Sem Redis, tudo em memória.
- Limites: 20 conexões por IP (ou rede `/64` no IPv6) por instância, mensagem do cliente até 4 KB, ping a cada 30 s. O público só abre para código válido e conta no rate limit do rastreio.

## Ver também

Decisões [058](../../docs/decisoes/058-tempo-real-por-websocket-coder-websocket-servidor.md) a [062](../../docs/decisoes/062-com-redis-url-e-o-redis-fora.md), [086](../../docs/decisoes/086-rate-limit-e-limite-de-websocket-por.md). Segurança: item 22.
