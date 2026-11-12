# internal/database: migrations e queries

`migrations/` tem o SQL versionado (up/down), embutido com `go:embed` e aplicado quando a API sobe. `queries/` é a fonte do sqlc, que gera `internal/store`.

## Regras

- SQL só em `queries/*.sql`, sempre com parâmetros. Filtro opcional com `sqlc.narg`, nunca concatenando string.
- `internal/store` é gerado: mude a query e rode `make sqlc` (na nuvem, `sqlc generate` depois do `go install`). Nunca edite o gerado; a CI confere se está em dia.
- Mudança de schema = migration nova com `up` e `down`. Migration que já está na `main` nunca é editada.
- `CHECK` no banco garante status e papel válidos e latitude/longitude juntas, não só a aplicação.
- Tabelas que guardam dado pessoal entram na retenção (`internal/retention`) e no item 27 de [SECURITY](../../docs/security/dados.md).

## Armadilhas conhecidas

- **air aplica migration na hora:** com `docker compose up` rodando, salvar um `.sql` em `migrations/` recompila a API, que migra o banco de desenvolvimento. Escreva a migration inteira antes de salvar, ou pare o container.
- **`timestamptz` que aceita nulo** vira `pgtype.Timestamptz` sem o override para `*time.Time` que está no `sqlc.yaml`.

## Ver também

Decisões [006](../../docs/decisoes/006-sqlc-pgx-em-vez-de-orm.md), [007](../../docs/decisoes/007-migrations-embutidas-e-aplicadas-ao-subir-a.md), [026](../../docs/decisoes/026-sqlc-pela-imagem-docker-oficial-make-sqlc.md).
