# 071. Sem período de tolerância para refresh simultâneo

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Mais simples e mais seguro; o front garante uma renovação por vez. Duas abas renovando ao mesmo tempo podem derrubar a sessão: se virar problema, entra uma tolerância de poucos segundos.

## Decisão

Sem período de tolerância para refresh simultâneo.

## Consequência

Em 07/10 o problema apareceu e foi resolvido no front, com a trava entre abas (ver acima).
