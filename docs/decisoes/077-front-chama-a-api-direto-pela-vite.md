# 077. Front chama a API direto pela `VITE_API_URL`

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O CORS já aceita `http://localhost:5173`; em produção o front fica na CDN e a API em outro domínio.

## Decisão

Front chama a API direto pela `VITE_API_URL`.

Alternativas descartadas: proxy do Vite.

## Consequência

Não registrada além do contexto.
