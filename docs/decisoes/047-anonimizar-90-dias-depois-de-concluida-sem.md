# 047. Anonimizar 90 dias depois de concluída, sem apagar a entrega

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O link público já expira em 30; a transportadora ainda precisa de alguns meses para reclamações, e os relatórios continuam contando entregas e status. Roda na própria API a cada hora: o `UPDATE` só pega linhas não anonimizadas, então várias instâncias podem rodar juntas.

## Decisão

Anonimizar 90 dias depois de concluída, sem apagar a entrega.

Alternativas descartadas: apagar a linha, que quebra relatórios; job separado ou cron, mais uma peça para subir.

## Consequência

Não registrada além do contexto.
