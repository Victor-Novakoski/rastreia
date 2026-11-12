# internal/route: rota do dia do motorista

## Regras

- A rota é do motorista e do dia, no fuso de São Paulo; amanhã começa vazia.
- Bipar (`POST /me/route/deliveries`) aceita o código ou o link do QR-code. Pacote sem motorista passa a ser de quem bipou; de outro motorista responde 409; de outra transportadora, 404. Ler o mesmo QR duas vezes não duplica.
- Parada = mesmo CEP, rua e número, sem olhar o complemento. Endereço sem número só se junta a outro no mesmo ponto do mapa.
- A ordem é guardada por pacote em `route_items`; as paradas são montadas na leitura.
- Ordem sugerida (`POST /me/route/optimize`): vizinho mais próximo + 2-opt em linha reta, a partir de onde o motorista está. Sem serviço de rotas externo.

## Ver também

Decisões [031](../../docs/decisoes/031-rota-e-do-motorista-e-do-dia.md), [032](../../docs/decisoes/032-bipar-pacote-sem-motorista-atribui-ao-motorista.md), [033](../../docs/decisoes/033-parada-mesmo-cep-rua-e-numero-sem.md), [038](../../docs/decisoes/038-ordem-sugerida-por-vizinho-mais-proximo-2.md), [039](../../docs/decisoes/039-a-ordem-e-guardada-por-pacote-as.md), [091](../../docs/decisoes/091-endereco-sem-numero-so-vira-a-mesma.md).
