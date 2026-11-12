# 055. Endpoint de push só em hosts dos serviços dos navegadores

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O worker faz POST na URL que o navegador manda; sem a lista, seria SSRF ([SECURITY.md](../SECURITY.md) #17).

## Decisão

Endpoint de push só em hosts dos serviços dos navegadores.

Alternativas descartadas: bloquear só IPs privados, que exige checar depois do DNS.

## Consequência

Não registrada além do contexto.
