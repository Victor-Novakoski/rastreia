# 040. Várias transportadoras, cada uma um tenant, com `carrier_id` em usuários e entregas

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O "admin" único não fazia sentido como produto; com o tenant, uma transportadora se cadastra e começa a usar sozinha.

## Decisão

Várias transportadoras, cada uma um tenant, com `carrier_id` em usuários e entregas.

Alternativas descartadas: banco ou schema por transportadora, pesado demais para o tamanho do projeto; Row Level Security do Postgres, que esconde a regra fora do código e complica os testes.

## Consequência

Não registrada além do contexto.
