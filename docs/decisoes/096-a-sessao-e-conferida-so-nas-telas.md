# 096. A sessão é conferida só nas telas que dependem dela

- **Data:** 2026-10-08
- **Status:** ativa

## Contexto

Quem abre o rastreio recebe a entrega e não tem conta; perguntar pela sessão a cada visita gastava uma requisição, deixava um aviso no log e consumia as renovações do IP, que numa rede compartilhada são as de quem está logado. A página inicial continua perguntando, para levar quem já entrou direto à sua área.

## Decisão

A sessão é conferida só nas telas que dependem dela.

Alternativas descartadas: a sessão conferida na aplicação inteira; um marcador no `localStorage` dizendo se existe sessão, que pode mentir quando o cookie vence.

## Consequência

Não registrada além do contexto.
