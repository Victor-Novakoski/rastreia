# 001. Conventional Commits e CI no GitHub Actions

- **Data:** 2026-10-01
- **Status:** ativa

## Contexto

Histórico legível e todo PR conferido (lint, sqlc, testes, imagem, govulncheck, gitleaks, título do PR).

## Decisão

Conventional Commits e CI no GitHub Actions.

## Consequência

Actions fixadas por hash do commit. Depois entraram o job do front, o Trivy e o npm audit.
