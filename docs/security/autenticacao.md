# Segurança: autenticação e sessão

Itens 5, 7, 8, 10, 14, 18 e 24: login, senhas, força bruta, tokens, cookies, CSRF e enumeração de e-mails. Resumo e legenda em [SECURITY.md](../SECURITY.md).

## 5. Autenticação fraca — ✅

**Feito**
- Senhas com bcrypt; mínimo de 10 caracteres, máximo de 72 bytes e recusa de senhas comuns (lista em `internal/auth/policy.go`).
- Mesma mensagem e mesmo tempo de resposta para e-mail inexistente e senha errada (item 24).
- JWT com algoritmo fixo (HS256), validade obrigatória e papel validado ao ler o token.
- O papel vem do banco no login, nunca do corpo da requisição.
- Senha montada a partir do e-mail é recusada: a parte antes do `@`, sem diferenciar maiúsculas nem pontuação, não pode aparecer na senha (`joao.silva@...` não aceita uma senha que contenha "joaosilva"). Partes com menos de 4 letras ficam de fora, porque bateriam com senhas demais.
- Em produção a API não sobe com a senha padrão da conta de demonstração.
- Ver também tokens (14).

## 7. Senhas no banco — ✅

- Só o hash bcrypt é salvo (`password_hash`), nunca a senha.
- O tipo `user.User`, que é o que a API devolve, não tem o campo do hash.
- Senha nunca vai para log.
- Futuro: se o custo do bcrypt mudar, refazer o hash no próximo login.

## 8. Ataque de força bruta — ✅

- Limite de 10 tentativas de login por minuto por IP, com 429 e `Retry-After`.
- Bloqueio por e-mail: depois de 5 senhas erradas, o e-mail fica bloqueado por 1 minuto, e o tempo dobra a cada nova falha até 15 minutos. Vale também para e-mails que não existem, para o bloqueio não revelar quem tem conta. Senha certa zera o contador.
- Cada falha é registrada no log com o IP e uma impressão do e-mail (não o e-mail em si).
- Com `REDIS_URL`, o contador fica no Redis e vale para todas as instâncias, numa operação atômica (script Lua). As chaves levam um hash SHA-256 do e-mail, nunca o e-mail. Se o Redis não responde, o login devolve 503 em vez de seguir sem proteção.

## 10. CSRF — ✅

As rotas que alteram dados usam `Authorization: Bearer`, que o navegador não envia sozinho. Só `/auth/refresh` e `/auth/logout` dependem do cookie, e elas têm duas barreiras: o cookie é `SameSite=Strict` e restrito a `Path=/auth`, e a requisição precisa de um cabeçalho `Origin` da lista `CORS_ORIGINS` (sem `Origin`, 403).

`/auth/login` e `/auth/signup` também recusam (403) um `Origin` fora da lista: outro site não consegue logar o navegador da vítima numa conta do atacante (*login CSRF*). Clientes que não são navegador, como o curl, não mandam `Origin` e passam.

Se algum dia a sessão inteira for por cookie, token anti-CSRF obrigatório nas rotas que alteram dados.

## 14. Tokens mal otimizados — ✅

**Feito**
- Access token JWT de 15 min (`JWT_TTL`), com algoritmo fixo (HS256), claims mínimos (`sub`, `role`, `cid`, `iat`, `exp`) e `iss`/`aud` validados.
- Refresh token opaco (32 bytes aleatórios) no cookie `rastreia_refresh`, salvo no banco **só como hash** (SHA-256), válido por `REFRESH_TTL` (7 dias) sem uso.
- Rotação: cada uso marca o token como usado e entrega o próximo da mesma família. Reuso de um token já trocado revoga a família inteira (sinal de roubo). Dois refreshes simultâneos com o mesmo token também contam como reuso.
- `POST /auth/logout` revoga a sessão.
- No front, o access token fica só em memória (item 20), e a renovação passa por uma trava entre abas (`navigator.locks`), para duas abas não usarem o mesmo refresh token e derrubarem a sessão.

**Falta:** revogar as sessões ao trocar a senha ou desativar o usuário, quando essas funções existirem (issue [#53](https://github.com/Victor-Novakoski/rastreia/issues/53)).

## 18. Cookies inseguros — ✅

O único cookie é o `rastreia_refresh`: `HttpOnly`, `SameSite=Strict`, `Path=/auth`, `Secure` sempre (Chrome e Firefox aceitam cookie `Secure` em `http://localhost`, então o desenvolvimento não precisa de HTTPS) e validade igual à do refresh token. Não dá para usar o prefixo `__Host-`, que exige `Path=/`.

**Requisito de deploy:** front e API no mesmo site (ex.: `app.dominio` e `api.dominio`), senão o `SameSite=Strict` impede o envio do cookie.

## 24. Enumeração de e-mails — ✅

Quando o e-mail não existe, o login compara a senha com um hash bcrypt fixo, então o tempo é o mesmo dos dois jeitos (medido: ~80 a 90 ms em ambos). A mensagem também é a mesma, e o bloqueio por falhas vale para qualquer e-mail. O cadastro de motorista (409 `e-mail already in use`) só é acessível à transportadora. O cadastro aberto de transportadora também responde 409 para e-mail usado, o que revela que o e-mail tem conta; é o mesmo trade-off de qualquer cadastro aberto, e a rota tem o mesmo limite do login (10/min por IP), com contador próprio.
