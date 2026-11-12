# 093. Log de requisição próprio, em JSON pelo `slog`, no lugar do `middleware.Logger` do chi

- **Data:** 2026-10-07
- **Status:** ativa

## Contexto

Uma linha por requisição no mesmo formato dos outros logs, que dá para pesquisar; 401, 403 e 429 viram aviso com o motivo, e os middlewares de autenticação acrescentam quem fez pela `httpx.AddLogAttrs`. A rota entra no lugar do caminho para o log não guardar códigos de rastreio nem buscas.

## Decisão

Log de requisição próprio, em JSON pelo `slog`, no lugar do `middleware.Logger` do chi.

Alternativas descartadas: Logger do chi, que escreve texto e o caminho com a query; um log separado em cada middleware que nega.

## Consequência

Não registrada além do contexto.
