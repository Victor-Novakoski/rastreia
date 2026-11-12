# internal/auth: login, tokens e papéis

## Regras

- JWT HS256 com `sub`, `role` e `cid` (transportadora), 15 min (`JWT_TTL`), `iss`/`aud` validados. `JWT_SECRET` com 32+ caracteres; em produção a API recusa o valor de exemplo.
- Refresh token opaco, guardado só como hash SHA-256, com rotação e família: reuso revoga a família inteira. Não há período de tolerância; quem evita duas renovações juntas é o front (trava entre abas).
- Cookie `rastreia_refresh`: `HttpOnly`, `SameSite=Strict`, `Path=/auth`, mais checagem de `Origin` em login, cadastro e refresh. Exige front e API no mesmo site.
- Papéis: `carrier` e `driver`. Token antigo sem `cid` é recusado.
- Login: tempo constante para e-mail inexistente (hash fixo), mesma mensagem, bloqueio progressivo por e-mail depois de 5 erros contando e-mails que não existem. Log de falha leva só uma impressão do e-mail.
- Senha: 10 a 72 bytes, recusa senhas comuns e a parte do e-mail antes do `@` (com 4 letras ou mais).
- Com `REDIS_URL` e o Redis fora do ar, o login responde 503 em vez de logar sem contador.

## Armadilhas conhecidas

- **bcrypt aceita no máximo 72 bytes** e dá erro acima disso; a validação recusa antes, com 422.
- **A senha do admin de teste está na lista de senhas comuns,** mas é aceita só para o admin criado pela configuração. Ela é definida em `ADMIN_PASSWORD` (`.env.example` e `docker-compose.yml`); em produção a API recusa esse valor (`internal/config`).

## Ver também

Decisões [018](../../docs/decisoes/018-bloqueio-de-login-conta-e-mails-inexistentes.md), [070](../../docs/decisoes/070-refresh-token-opaco-no-banco-hash-sha.md), [071](../../docs/decisoes/071-sem-periodo-de-tolerancia-para-refresh-simultaneo.md), [072](../../docs/decisoes/072-cookie-rastreia-refresh-com-path-auth-e.md), [084](../../docs/decisoes/084-login-e-cadastro-tambem-conferem-o-origin.md), [085](../../docs/decisoes/085-a-senha-nao-pode-conter-a-parte.md). Segurança: itens 5, 8, 14, 18 e 24.
