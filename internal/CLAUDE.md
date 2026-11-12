# internal: back-end em Go

Um pacote por domínio (`user`, `delivery`, `route`, `notify`, `push`, `retention`, `realtime`) mais os de apoio (`auth`, `config`, `database`, `store`, `apperr`, `httpx`, `server`, `testdb`, `testredis`). Visão das camadas em [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md).

## Regras

- **Camadas:** handler só fala HTTP (`httpx.Decode`, `httpx.JSON`, `httpx.WriteError`); regra de negócio no service; banco só pelo sqlc.
- **Store:** interface pequena no próprio pacote do service, só com os métodos que ele usa; nos testes vira um fake em memória.
- **Transação:** operação que escreve em mais de uma tabela usa `Store.InTx` (implementado em `delivery` e `route` com `pgx.BeginFunc`).
- **Tipos do domínio:** o service nunca devolve a struct do `store` ao handler (ex.: `user.User` não carrega o hash da senha).
- **Erros:** `apperr.Validator` para validação (422 com os campos), `apperr.ErrNotFound`/`ErrConflict` para os casos conhecidos. Erro inesperado sobe com `fmt.Errorf("contexto: %w", err)` e vira 500 genérico.
- **Tenant:** o `carrier_id` vem do `auth.Claims`, nunca do corpo. Listagens filtram na query; busca por id confere no service (`visible`) e responde 404 quando não bate. Os testes de IDOR cobrem isso.
- **Limites de texto** contam caracteres, não bytes ("São" tem 4 bytes).
- **Testes:** services com fakes do `Store`, handlers com `httptest`, integração com Postgres e Redis reais ([testdb/CLAUDE.md](testdb/CLAUDE.md)).

## Ver também

Decisões [005](../docs/decisoes/005-go-com-chi-sem-framework.md), [006](../docs/decisoes/006-sqlc-pgx-em-vez-de-orm.md), [041](../docs/decisoes/041-carrier-id-no-token-cid-e-filtro.md), [090](../docs/decisoes/090-limites-de-texto-em-caracteres-nao-em.md).
