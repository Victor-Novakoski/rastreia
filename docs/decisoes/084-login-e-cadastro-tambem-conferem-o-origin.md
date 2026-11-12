# 084. Login e cadastro também conferem o `Origin`

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

Sem isso, outro site poderia logar o navegador da vítima numa conta do atacante (login CSRF). Clientes que não são navegador não mandam `Origin` e passam.

## Decisão

Login e cadastro também conferem o `Origin`.

Alternativas descartadas: token anti-CSRF, que exigiria uma chamada a mais antes do login.

## Consequência

Não registrada além do contexto.
