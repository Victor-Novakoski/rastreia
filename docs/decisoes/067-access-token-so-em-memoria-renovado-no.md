# 067. Access token só em memória, renovado no 401

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Recarregar a página chama `/auth/refresh` com o cookie; um 401 renova uma vez e repete a chamada. Os refreshes simultâneos dividem a mesma requisição, porque usar o mesmo refresh token duas vezes derruba a sessão.

## Decisão

Access token só em memória, renovado no 401.

Alternativas descartadas: `localStorage`, que o XSS lê; renovar por timer.

## Consequência

Não registrada além do contexto.
