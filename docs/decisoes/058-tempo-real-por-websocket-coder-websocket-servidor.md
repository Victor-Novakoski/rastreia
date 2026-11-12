# 058. Tempo real por WebSocket (`coder/websocket`), servidor só envia

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O rastreio público recebe o mesmo corpo do `GET`, então a página não refaz a consulta; o painel recebe só `{delivery_id, status}` e invalida o cache do TanStack Query, sem dado pessoal no fio.

## Decisão

Tempo real por WebSocket (`coder/websocket`), servidor só envia.

Alternativas descartadas: SSE, que serviria e é mais simples, mas o PRD pede WebSocket; polling.

## Consequência

Não registrada além do contexto.
