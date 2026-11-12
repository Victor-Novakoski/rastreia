# 070. Refresh token opaco no banco (hash SHA-256), com rotação e família

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Dá para revogar (logout, reuso) e o banco vazado não entrega tokens usáveis; reuso derruba a família inteira.

## Decisão

Refresh token opaco no banco (hash SHA-256), com rotação e família.

Alternativas descartadas: refresh em JWT, sem estado: não dá para revogar.

## Consequência

Não registrada além do contexto.
