# 094. Busca do painel com `LIKE`, e os acentos tirados por `translate` no SQL e por um `Replacer` no Go

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

"joao" acha "João" sem instalar extensão no Postgres; a busca roda só sobre as entregas da transportadora, que o índice de `carrier_id` já separa, então não precisa de índice próprio.

## Decisão

Busca do painel com `LIKE`, e os acentos tirados por `translate` no SQL e por um `Replacer` no Go.

Alternativas descartadas: pg_trgm com índice GIN e unaccent, extensões a instalar no banco; busca no front, que só enxerga a página carregada.

## Consequência

Não registrada além do contexto.
