# 012. Hot reload com air via `docker-compose.override.yml`

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

`docker compose up` já sobe o ambiente de desenvolvimento, sem make nem `-f`; a imagem de produção continua sendo o estágio final do Dockerfile.

## Decisão

Hot reload com air via `docker-compose.override.yml`.

Alternativas descartadas: `make dev`, arquivo `docker-compose.dev.yml`.

## Consequência

Não registrada além do contexto.
