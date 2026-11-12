# 019. Concorrência otimista na troca de status

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

O `UPDATE` confere o status anterior (`WHERE status = from_status`); dois eventos simultâneos não se sobrepõem e o perdedor recebe 409, sem lock explícito.

## Decisão

Concorrência otimista na troca de status.

Alternativas descartadas: `SELECT ... FOR UPDATE`.

## Consequência

Não registrada além do contexto.
