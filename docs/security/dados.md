# Segurança: dados pessoais e logs

Itens 12, 16, 26 e 27: vazamento de informação, dados sensíveis, logs e auditoria, LGPD e retenção. Resumo e legenda em [SECURITY.md](../SECURITY.md).

## 12. Vazamento de informação — 🟡

**Feito**
- Erro inesperado vira `500 {"error":"internal error"}`; o detalhe só vai para o log.
- Pânico é capturado por um middleware que responde 500 e loga a pilha, sem stack trace na resposta.
- Rota que não existe e método errado respondem no mesmo JSON dos outros erros, sem o texto padrão do roteador.
- Login não diz se o e-mail existe, nem pela mensagem nem pelo tempo de resposta (item 24).
- Headers de segurança em todas as respostas (item 21).

**Falta**
- Decidir se `/openapi.yaml` continua público em produção (hoje é; não expõe segredo, mas mapeia a API). Issue [#50](https://github.com/Victor-Novakoski/rastreia/issues/50).

## 16. Dados sensíveis expostos — 🟡

**Feito**
- Hash de senha nunca sai da API; `.env` fora do git; logs não registram corpo de requisição nem token.
- O rastreio público tem resposta própria (`delivery.Tracking`): código, nome da transportadora, status, histórico (status e horário) e primeiro nome do destinatário. **Não** inclui e-mail, telefone, endereço, coordenadas, sobrenome, motorista, ids internos nem as observações dos eventos (texto livre do motorista pode ter dado pessoal). Testado no serviço e pela API.
- Com `APP_ENV=production`, toda resposta leva `Strict-Transport-Security` (item 21).

**Falta**
- HTTPS no deploy (issue [#44](https://github.com/Victor-Novakoski/rastreia/issues/44)).
- Banco exposto (item 25).

## 26. Logs e auditoria — 🟡

**Feito**
- Todo log sai em JSON pelo `slog`, com uma linha por requisição (`internal/server/log.go`): método, rota, status, duração, IP, request ID e, nas rotas autenticadas, quem fez (`user_id` e `carrier_id`).
- 401, 403 e 429 saem como aviso, com a mensagem `request denied` e o motivo (`missing token`, `invalid token`, `role driver not allowed`, `rate limit login`, `login locked`, `wrong e-mail or password`). Dá para filtrar por eles e criar alertas. Erros 5xx saem como erro, e pânico vem com a pilha.
- Falha de login leva o IP e uma impressão do e-mail (SHA-256 cortado), nunca o e-mail.
- O log grava a rota (`/public/tracking/{code}`), não o caminho, e nunca a query string: código de rastreio e a busca do painel (que pode ter o nome do destinatário) ficam de fora. Sem senha, token ou corpo de requisição no log. Testado em `internal/server`.
- WebSocket do painel recusado (token inválido na primeira mensagem) também gera um aviso, `websocket denied`.

**Falta**
- Auditoria de negócio da edição de entregas (`PATCH`), issue [#49](https://github.com/Victor-Novakoski/rastreia/issues/49). Mudanças de status já ficam em `delivery_events`, com quem fez e quando.
- Alertas em produção para pico de falhas de login e de 5xx (issue [#48](https://github.com/Victor-Novakoski/rastreia/issues/48)). Antes deles, separar o `/auth/refresh` sem cookie (visitante da página inicial, que é normal) do refresh com token inválido ou reusado: hoje os dois saem como o mesmo aviso (issue [#47](https://github.com/Victor-Novakoski/rastreia/issues/47)).

## 27. LGPD e retenção de dados — ✅

O sistema guarda nome, e-mail, telefone e endereço do destinatário, com o ponto de referência e as coordenadas no mapa.

**Feito**
- Rastreio público sem dados pessoais além do primeiro nome (item 16).
- O link público deixa de funcionar 30 dias depois de a entrega ser entregue ou da última falha (responde 404, como um código inexistente).
- E-mail e push de notificação levam só o primeiro nome, o código e o status, sem endereço nem observação do motorista. O nome passa pelo `html/template`, que escapa HTML.
- As inscrições de push são apagadas quando a entrega é entregue ou quando o serviço de push diz que expiraram.
- Retenção: `RETENTION_DAYS` (padrão 90, mínimo 31) dias depois de entregue ou da última falha, a API troca o nome por "Destinatário removido" e apaga e-mail, telefone, endereço (CEP, rua, número, complemento, bairro e ponto de referência), coordenadas, as observações dos eventos (texto livre que pode citar pessoas) e as inscrições de push. Entrega que ficou parada mais de 1 ano sem terminar recebe o mesmo tratamento. Ficam a cidade, a UF, a entrega e o histórico de status, para os relatórios; a entrega anonimizada não aceita mais alteração (409), para os dados não voltarem, e o link público dela responde 404. Roda a cada hora em `internal/retention`, testado com Postgres real.
- A mesma tarefa apaga as chaves de idempotência com mais de 24 horas e os refresh tokens vencidos.
- Coleta só o necessário: o telefone entra porque o motorista precisa falar com o destinatário na entrega; CPF e outros documentos do destinatário não são pedidos.

**Falta**
- Backups seguem a mesma regra quando existirem (issue [#46](https://github.com/Victor-Novakoski/rastreia/issues/46)).
