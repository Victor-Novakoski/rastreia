# 025. Testes de integração com testcontainers, no mesmo `go test`

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

Um container por pacote e um banco novo por teste; pulados com `-short` ou sem Docker, então `make test-short` roda sem Docker.

## Decisão

Testes de integração com testcontainers, no mesmo `go test`.

Alternativas descartadas: build tag `integration`.

## Consequência

Não registrada além do contexto.
