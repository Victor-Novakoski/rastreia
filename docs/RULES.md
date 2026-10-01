# Regras do projeto

Regras que valem para qualquer mudança. Se uma regra atrapalhar, ela é discutida e alterada aqui, não ignorada em silêncio.

## 1. Escopo

- Toda funcionalidade nova precisa estar no [PRD](PRD.md) e em [TASKS.md](TASKS.md) antes de ser implementada.
- Uma tarefa por vez. Melhorias que aparecerem no caminho viram item novo em TASKS, não entram de carona.
- Nada de dependência, serviço ou camada nova "para o futuro". Entra quando uma tarefa precisar.

## 2. Código Go

- Respeitar as camadas de [ARCHITECTURE.md](ARCHITECTURE.md): handler só fala HTTP; regra de negócio fica no service; banco só via sqlc.
- SQL só em `internal/database/queries/*.sql`, sempre com parâmetros. Nunca montar SQL concatenando strings.
- `internal/store` é gerado: alterar a query e rodar `make sqlc`, nunca editar o arquivo gerado.
- Mudança de schema = nova migration (`up` e `down`). Migration que já está na `main` não é editada.
- Services devolvem tipos do domínio, nunca a struct do `store` direto para o handler (evita vazar campos como `password_hash`).
- Erros: `apperr.Validator` para validação, `apperr.ErrNotFound`/`ErrConflict` para os casos conhecidos. Erro inesperado sobe com `fmt.Errorf("contexto: %w", err)` e vira 500 genérico.
- Comentários e identificadores em inglês, como o código atual. Docs, README e mensagens de commit em português.
- `gofmt` e `go vet` limpos (`make lint`).

## 3. Segurança

Checklist para toda mudança (detalhes em [SECURITY.md](SECURITY.md)):

- [ ] Entrada validada no back-end, com tamanho máximo para textos, mesmo que o front já valide.
- [ ] Rota nova tem autenticação e papel definidos explicitamente; rota pública é exceção justificada.
- [ ] Recurso acessado por id confere se pertence a quem pede (IDOR).
- [ ] Resposta não expõe hash, token, dados pessoais desnecessários nem detalhes internos de erro.
- [ ] Nenhum segredo no código, em log ou em commit. Valores novos vão para `.env.example` com placeholder.
- [ ] Dependência nova é necessária, mantida e passa no `govulncheck`.

## 4. Testes

- Regra de negócio nova tem teste no service. Rota nova tem teste do handler (status e corpo).
- Bug corrigido ganha teste que falhava antes da correção.
- `make test` passando antes de qualquer commit.

## 5. API

- Seguir as convenções de [DESIGN.md](DESIGN.md) (formato de erro, paginação, nomes em snake_case).
- Toda rota nova ou alterada é atualizada em `api/openapi.yaml` no mesmo commit.

## 6. Git

- **Branches:** `main` é o que está publicado; `develop` junta o trabalho pronto para a próxima versão. Ninguém faz push direto nas duas: tudo entra por pull request.
- **Fluxo:** criar a branch a partir da `develop` (`feat/...`, `fix/...`, `docs/...`, `chore/...`), abrir PR para a `develop` e fazer merge com os checks verdes. A branch é apagada automaticamente depois do merge.
- **Versão:** quando a `develop` fecha uma etapa, abrir PR da `develop` para a `main`.
- `main` e `develop` sempre funcionando: sobem com `docker compose up` e passam nos testes.
- Commits pequenos, com mensagem em português que diz o que muda (ex.: "Rate limit no login").
- Nunca commitar `.env`, binários ou arquivos gerados fora do sqlc.

## 7. Documentação

- Mudou comportamento, rota, variável de ambiente ou decisão de arquitetura: atualizar o doc correspondente no mesmo commit.
- Decisão relevante (escolha de lib, trade-off, algo que foi descartado) vai para [MEMORY.md](MEMORY.md).
- Tarefa concluída é marcada em [TASKS.md](TASKS.md).

## Definição de pronto

Uma tarefa só está pronta quando: funciona com `docker compose up`, tem testes, passa no lint, o checklist de segurança foi revisado, OpenAPI e docs estão atualizados e o item está marcado em TASKS.
