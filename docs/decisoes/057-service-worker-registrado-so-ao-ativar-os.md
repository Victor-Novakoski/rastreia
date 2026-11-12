# 057. Service worker registrado só ao ativar os avisos

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Quem não usa push não ganha um service worker; o front não tem cache offline.

## Decisão

Service worker registrado só ao ativar os avisos.

Alternativas descartadas: vite-plugin-pwa, que traria cache e mais configuração.

## Consequência

Não registrada além do contexto.
