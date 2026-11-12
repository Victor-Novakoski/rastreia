# internal/delivery: entregas, eventos e rastreio público

## Regras

- **Status muda só por evento** (`POST /deliveries/{id}/events`), nunca por `PATCH`. Criar a entrega já grava o evento `pending`.
- **Transições:** `failed` exige observação e pode voltar para `in_transit`; `delivered` é final.
- **Concorrência otimista:** o `UPDATE` confere o status lido (`WHERE status = from_status`); quem perde recebe 409.
- **Anonimizada não muda:** as queries exigem `anonymized_at IS NULL`; alteração responde 409 e o rastreio dela, 404.
- **Rastreio público** (`/public/tracking/{code}`) tem resposta própria (`Tracking`): sem e-mail, telefone, endereço, coordenadas, sobrenome, motorista, ids nem observações. Código malformado, inexistente e expirado (30 dias depois de concluída) respondem o mesmo 404.
- **Código de rastreio:** `RS` + 10 caracteres sem 0/O/1/I; o id sequencial nunca é público.
- **`Idempotency-Key`** no `POST /deliveries` guarda o id da entrega, reservada na mesma transação da criação.
- **Endereço** em colunas; `address` é montado pela API. Mudar o endereço sem mandar coordenadas apaga as antigas.
- **Busca** com `LIKE` e acentos tirados por `translate` no SQL e um `Replacer` no Go, sem extensão no Postgres.
- O motorista abre a própria entrega por `GET /deliveries/{id}`; `/me/deliveries` mostra primeiro o que falta e deixa de fora as anonimizadas.

## Ver também

Decisões [009](../../docs/decisoes/009-status-muda-so-por-evento-nunca-por.md), [019](../../docs/decisoes/019-concorrencia-otimista-na-troca-de-status.md), [022](../../docs/decisoes/022-rastreio-publico-sem-as-observacoes-dos-eventos.md), [024](../../docs/decisoes/024-idempotency-key-guarda-o-id-da-entrega.md), [087](../../docs/decisoes/087-entrega-parada-ha-mais-de-1-ano.md), [094](../../docs/decisoes/094-busca-do-painel-com-like-e-os.md). Retenção em `internal/retention` (90 dias, `RETENTION_DAYS`).
