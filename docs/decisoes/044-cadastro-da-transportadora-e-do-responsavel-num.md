# 044. Cadastro da transportadora e do responsável num único `INSERT` com CTE

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

É atômico sem precisar de transação no service: e-mail repetido não deixa transportadora sem dono.

## Decisão

Cadastro da transportadora e do responsável num único `INSERT` com CTE.

Alternativas descartadas: transação com `InTx` no pacote user.

## Consequência

Não registrada além do contexto.
