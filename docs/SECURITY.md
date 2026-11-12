# Segurança

Como o Rastreia trata cada risco, o que já está feito e o que falta. O que falta virou issue no GitHub; o checklist para cada mudança está no [CLAUDE.md](../CLAUDE.md#segurança-checklist-de-toda-mudança).

Situação revisada em 08/10/2026, ao fim da etapa 6.7 (acabamento e revisão geral).

**Legenda:** ✅ feito · 🟡 parcial · 🔴 pendente · ⚪ ainda não se aplica (regra definida para quando se aplicar)

## Resumo

| # | Risco | Situação | Prioridade | O que é |
| --- | --- | --- | --- | --- |
| 1 | Variáveis de ambiente expostas | 🟡 | Deploy | Senhas, chaves e segredos vazando por arquivo commitado, imagem Docker ou valor padrão usado em produção. |
| 2 | Validação no front-end | ✅ | — | Conferir os dados no formulário para dar retorno rápido ao usuário. Ajuda na experiência, mas não protege nada: dá para burlar. |
| 3 | Validação no back-end | ✅ | — | A API confere tipo, formato e tamanho de tudo que recebe. É a validação que realmente protege. |
| 4 | SQL Injection | ✅ | — | Texto enviado pelo usuário vira parte do comando SQL e consegue ler ou apagar dados do banco. |
| 5 | Autenticação fraca | ✅ | — | Senha fraca ou previsível, login que dá pistas, ou credencial padrão que nunca foi trocada. |
| 6 | IDOR | ✅ | — | Trocar o id na URL (ex.: /deliveries/2 por /deliveries/3) e acessar dado de outra pessoa. |
| 7 | Senhas no banco | ✅ | — | Guardar a senha como texto no banco: se o banco vazar, todas as senhas vazam junto. |
| 8 | Força bruta | ✅ | — | Tentar milhares de senhas seguidas até acertar. |
| 9 | Envio duplicado | ✅ | — | Clique duplo, rede lenta ou reenvio que cria o mesmo registro duas vezes. |
| 10 | CSRF | ✅ | — | Outro site faz o navegador do usuário logado enviar uma ação para a API sem ele perceber. |
| 11 | Upload sem validação | ⚪ | — | Arquivo malicioso disfarçado (ex.: script com extensão .jpg) ou grande demais para o servidor. |
| 12 | Vazamento de informação | 🟡 | Deploy | Mensagem de erro, stack trace ou cabeçalho que conta detalhes internos para um atacante. |
| 13 | Dependências vulneráveis | ✅ | — | Biblioteca de terceiros com falha de segurança conhecida. |
| 14 | Tokens | ✅ | — | Token que vale por muito tempo, não pode ser revogado ou carrega dados demais. |
| 15 | Rate limit | ✅ | — | Limitar quantas requisições cada cliente faz por minuto, contra abuso e força bruta. |
| 16 | Dados sensíveis expostos | 🟡 | Deploy (HTTPS) | Resposta da API, log ou link público mostrando dado pessoal ou secreto além do necessário. |
| 17 | SSRF | ✅ | — | Fazer o servidor chamar uma URL escolhida pelo atacante, como a rede interna ou os metadados da nuvem. |
| 18 | Cookies inseguros | ✅ | — | Cookie que o JavaScript pode ler, que trafega sem HTTPS ou que é enviado por outros sites. |
| 19 | CORS | ✅ | — | Regra do navegador que diz quais sites podem chamar a API. |
| 20 | XSS | ✅ | — | Script injetado num dado (ex.: no nome) que roda no navegador de quem abre a página. |
| 21 | Headers de segurança | ✅ | — | Cabeçalhos HTTP que mandam o navegador se proteger (ex.: não abrir a página dentro de iframe, só usar HTTPS). |
| 22 | Timeouts e negação de serviço | 🟡 | Deploy | Conexões lentas ou requisições enormes que prendem o servidor e derrubam a API. |
| 23 | Falsificação de IP | ✅ | — | Cliente mente o IP pelo cabeçalho X-Forwarded-For para escapar do rate limit. |
| 24 | Enumeração de e-mails | ✅ | — | Descobrir quais e-mails têm conta pela mensagem ou pelo tempo de resposta do login. |
| 25 | Banco de dados exposto | 🟡 | Deploy | Banco acessível pela rede ou pela internet, sem precisar passar pela API. |
| 26 | Logs e auditoria | 🟡 | Média | Registrar quem fez o quê e os eventos suspeitos, para investigar e criar alertas. |
| 27 | LGPD e retenção de dados | ✅ | — | Lei de proteção de dados: coletar só o necessário, mostrar o mínimo e apagar quando não precisar mais. |

Os itens 1 a 19 são a lista original; os itens 20 a 27 completam a cobertura.


## Detalhe por tema

- [Autenticação e sessão](security/autenticacao.md): itens 5, 7, 8, 10, 14, 18, 24. Login, senhas, força bruta, tokens, cookies, CSRF e enumeração de e-mails.
- [Aplicação e navegador](security/aplicacao.md): itens 2, 3, 4, 6, 9, 11, 19, 20, 21. Validação, SQL injection, IDOR, envio duplicado, upload, CORS, XSS e headers.
- [Dados pessoais e logs](security/dados.md): itens 12, 16, 26, 27. Vazamento de informação, dados sensíveis, logs e auditoria, LGPD e retenção.
- [Infraestrutura e deploy](security/infraestrutura.md): itens 1, 13, 15, 17, 22, 23, 25. Segredos, dependências, rate limit, SSRF, timeouts, IP do cliente e banco.
