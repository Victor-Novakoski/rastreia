# 072. Cookie `rastreia_refresh` com `Path=/auth` e `SameSite=Strict`, mais checagem de `Origin`

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O cookie só vai para as rotas de sessão, e o `Origin` fecha o CSRF. Exige front e API no mesmo site.

## Decisão

Cookie `rastreia_refresh` com `Path=/auth` e `SameSite=Strict`, mais checagem de `Origin`.

Alternativas descartadas: prefixo `__Host-`, que obriga `Path=/`.

## Consequência

Não registrada além do contexto.
