# Índice dos docs

Abra só o que a tarefa pede. Regras gerais, comandos e git estão no [CLAUDE.md](../CLAUDE.md) da raiz.

| Onde | Leia quando... |
|---|---|
| [PRD.md](PRD.md) | for criar ou mudar uma funcionalidade e precisar saber se ela cabe no produto, quem usa e o que está fora de escopo |
| [DESIGN.md](DESIGN.md) | for criar uma rota (formato de erro, paginação, nomes) ou uma tela (paleta, tipografia, componentes, status) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | precisar entender como API, worker, Redis, RabbitMQ e front se ligam antes de mexer em mais de uma peça |
| [SECURITY.md](SECURITY.md) (resumo) e [security/](security/) (por tema: autenticação, aplicação, dados, infraestrutura) | mexer em autenticação, dados pessoais, rate limit, CORS, CSP, log ou deploy, ou revisar se um risco já foi tratado |
| [decisoes/](decisoes/) | quiser saber por que algo é como é, ou antes de trocar uma biblioteca ou um desenho já escolhido |
| [api/openapi.yaml](../api/openapi.yaml) | criar ou mudar uma rota (atualize no mesmo commit) |
| [internal/CLAUDE.md](../internal/CLAUDE.md) | escrever qualquer código Go: camadas, erros, tenant, testes |
| [internal/database/CLAUDE.md](../internal/database/CLAUDE.md) | criar migration, mudar query ou regenerar o sqlc |
| [internal/auth/CLAUDE.md](../internal/auth/CLAUDE.md) | mexer em login, cadastro, senha, tokens, cookie ou papéis |
| [internal/delivery/CLAUDE.md](../internal/delivery/CLAUDE.md) | mexer em entregas, status, eventos, busca, idempotência ou rastreio público |
| [internal/route/CLAUDE.md](../internal/route/CLAUDE.md) | mexer na rota do motorista: bipar, paradas, ordem |
| [internal/notify/CLAUDE.md](../internal/notify/CLAUDE.md) | mexer em e-mail, Web Push, outbox, filas ou no worker |
| [internal/realtime/CLAUDE.md](../internal/realtime/CLAUDE.md) | mexer em WebSocket ou no pub/sub do Redis |
| [internal/server/CLAUDE.md](../internal/server/CLAUDE.md) | mexer em middlewares, rate limit, IP do cliente ou no log de requisições |
| [internal/testdb/CLAUDE.md](../internal/testdb/CLAUDE.md) | for rodar os testes de integração, principalmente numa sessão sem Docker |
| [web/CLAUDE.md](../web/CLAUDE.md) | mexer no front: sessão, CSP, mapa, QR-code, arraste, Playwright |
| [Issues](https://github.com/Victor-Novakoski/rastreia/issues) | quiser saber o que está aberto ou escolher a próxima tarefa |
