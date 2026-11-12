# internal/testdb (e internal/testredis): bancos para testes de integração

`testdb.New` sobe um Postgres por pacote de teste (testcontainers) e cria um banco novo, já migrado, para cada teste. `testredis` faz o mesmo com os bancos lógicos do Redis. Sem Docker ou com `-short`, os testes de integração são pulados.

## Rodar na nuvem (sem Docker)

- Postgres 16 está em `/usr/lib/postgresql/16`: `initdb` como o usuário `postgres` num diretório em `/var/tmp` e suba o servidor. `redis-server` também está instalado.
- Altere **temporariamente** o `start()` de `testdb.go` e `testredis.go` para ler uma URL local em vez de subir o container. Não commite essa mudança.
- Rode um pacote por vez (`go test ./internal/delivery/...`) e apague os bancos `test_*` entre as rodadas.
- No fim, mate os processos que sobraram com `pkill -x` ou `kill <pid>`, nunca `pkill -f` (ver CLAUDE.md raiz).

## Ver também

Decisão [025](../../docs/decisoes/025-testes-de-integracao-com-testcontainers-no-mesmo.md).
