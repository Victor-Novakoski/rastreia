# 098. Docs no padrão CLAUDE.md, INDEX e decisões numeradas

- **Data:** 2026-11-12
- **Status:** ativa

## Contexto

Os docs estavam em sete arquivos grandes (PRD, ARCHITECTURE, RULES, DESIGN, TASKS, SECURITY, DECISIONS). DECISIONS e SECURITY passavam de 140 e 280 linhas, regras se repetiam entre RULES, SECURITY e ARCHITECTURE, e as tarefas viviam num arquivo que ficava velho.

## Decisão

Duas camadas. Para pessoas: README, PRD, DESIGN, uma ARCHITECTURE curta e o resumo de SECURITY. Para o agente: `CLAUDE.md` na raiz, `docs/INDEX.md` como roteador, um `CLAUDE.md` por módulo com regras e armadilhas, e uma decisão por arquivo em `docs/decisoes/`. Tarefas ficam nas issues do GitHub.

Alternativas descartadas: manter o formato antigo, que crescia sem limite.

## Consequência

RULES, TASKS e DECISIONS saem. Cada regra mora num lugar só. Decisão não se edita: uma nova substitui a antiga, que só muda de status. Doc com mais de ~150 linhas se divide.
