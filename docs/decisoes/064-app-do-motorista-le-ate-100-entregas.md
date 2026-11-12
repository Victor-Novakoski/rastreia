# 064. App do motorista lê até 100 entregas de `/me/deliveries` e acha a entrega na lista

- **Data:** 2026-10-02
- **Status:** [substituída por 078](078-o-motorista-abre-a-propria-entrega-por.md)

## Contexto

A API não tem `GET /me/deliveries/{id}`, e 100 cobre o dia de um motorista; entregas antigas somem da lista, o que não atrapalha.

## Decisão

App do motorista lê até 100 entregas de `/me/deliveries` e acha a entrega na lista.

Alternativas descartadas: rota nova na API.

## Consequência

Substituída em 07/10: a entrega abre direto por `GET /deliveries/{id}`.
