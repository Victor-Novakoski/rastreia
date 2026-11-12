# 063. CSP do front numa `<meta>` injetada no build

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O front vai ser estático numa CDN, e a política acompanha o HTML sem depender da configuração do servidor; só `frame-ancestors` precisa ir no cabeçalho.

## Decisão

CSP do front numa `<meta>` injetada no build.

Alternativas descartadas: CSP só no cabeçalho da CDN; nonce, que exige servidor.

## Consequência

Não registrada além do contexto.
