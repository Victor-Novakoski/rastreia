# PRD — Rastreia

Documento de requisitos do produto. Define **o que** o Rastreia é e **o que não é**. Qualquer funcionalidade nova precisa caber aqui antes de entrar em [TASKS.md](TASKS.md).

## Problema

Transportadoras pequenas e médias acompanham entregas por planilha e WhatsApp. O cliente final não sabe onde está o pedido e liga para perguntar; o motorista não tem um jeito simples de registrar o que aconteceu; a transportadora só descobre um problema quando o cliente reclama.

## Solução

Uma plataforma de rastreio de entregas com três pontas:

0. **Página inicial**: explica o produto e leva cada pessoa à sua área ("Sou transportadora", "Sou motorista", "Rastrear encomenda").
1. **Painel da transportadora**: a transportadora se cadastra sozinha, cadastra motoristas e entregas, atribui entregas e acompanha tudo. Cada transportadora só vê os próprios dados.
2. **App do motorista** (web mobile): vê as próprias entregas e atualiza o status na rua, com poucos toques.
3. **Link público de rastreio** (cliente final): abre pelo código de rastreio, sem login, e vê o status em tempo real.

## Personas

| Persona | Quem é | O que precisa |
| --- | --- | --- |
| Transportadora | Dono ou operador da transportadora | Começar a usar sem falar com ninguém; cadastrar e distribuir entregas rápido; ver o que está atrasado ou falhou |
| Motorista | Entregador, no celular, muitas vezes com sinal ruim | Lista do dia e botão grande para mudar status; nada de formulário longo |
| Cliente final | Quem vai receber | Saber em que pé está a entrega sem criar conta nem ligar |

## Ciclo de vida da entrega

```
pending ──► picked_up ──► in_transit ──► delivered
   │            │              │
   └────────────┴──────────────┴──────► failed
```

- `pending`: criada, aguardando coleta.
- `picked_up`: motorista coletou.
- `in_transit`: a caminho do destinatário.
- `delivered`: entregue (estado final).
- `failed`: não foi possível entregar, com motivo (estado final nesta versão).

O status só muda por **eventos** registrados pelo motorista (ou pela transportadora), nunca por edição direta. Cada evento guarda quem mudou, quando e uma observação opcional, formando o histórico que o cliente vê.

## Requisitos funcionais

### Etapa 1 — Base da API (feito)
- **RF01** Login com e-mail e senha para transportadora e motorista, devolvendo um token.
- **RF02** Transportadora cadastra e lista motoristas.
- **RF03** Transportadora cria, lista (com filtro por status e paginação), consulta e edita entregas.
- **RF04** Toda entrega recebe um código de rastreio aleatório e legível (ex.: `RS7K2M9QXA4P`).

### Etapa 2 — Eventos e rastreio público
- **RF05** Motorista lista apenas as entregas atribuídas a ele.
- **RF06** Motorista (ou transportadora) registra um evento de status seguindo as transições válidas do ciclo acima.
- **RF07** Consulta pública por código de rastreio, sem login, mostrando status e histórico, **sem dados pessoais** do destinatário (e-mail e endereço completo ficam de fora).

### Etapa 3 — Front-end
- **RF08** Painel web da transportadora.
- **RF09** App web mobile do motorista.
- **RF10** Página pública de rastreio.

### Etapas seguintes
- **RF11** Atualização em tempo real da página de rastreio e do painel (WebSocket).
- **RF12** Notificação por e-mail ao destinatário quando o status muda (fila).

### Produto (várias transportadoras)
- **RF13** Várias transportadoras na mesma instalação. Cada uma é um *tenant*: motoristas, entregas e números ficam isolados, e o que é de outra transportadora responde como inexistente.
- **RF14** Cadastro aberto de transportadora (nome, CNPJ opcional, responsável, e-mail e senha), que já entra logada.
- **RF15** Página inicial que apresenta o produto e leva a três entradas: transportadora (entrar ou criar conta), motorista (entrar) e rastreio por código.
- **RF16** Visão geral da transportadora: entregas por status nos últimos 30 dias e entregas sem motorista.
- **RF17** O rastreio público mostra o nome da transportadora que está entregando.

### Endereço e rota do motorista
- **RF18** Endereço da entrega em partes (CEP, rua, número, complemento, bairro, cidade, UF), com telefone do destinatário e ponto de referência. O CEP preenche rua, bairro e cidade, e o endereço vira um ponto no mapa que a transportadora pode corrigir.
- **RF19** A transportadora imprime uma etiqueta por entrega, com o código de rastreio em QR-code.
- **RF20** O motorista monta a rota do dia bipando o QR-code de cada pacote (ou digitando o código). Pacote sem motorista passa a ser dele; de outro motorista é recusado.
- **RF21** Pacotes no mesmo endereço viram uma parada. O sistema sugere uma ordem curta a partir de onde o motorista está e numera os pacotes de 1 a N; o motorista pode mudar a ordem como quiser.

## Requisitos não funcionais

- **RNF01 Segurança:** seguir [SECURITY.md](SECURITY.md). Nenhuma etapa é concluída com item crítico pendente.
- **RNF02 Privacidade (LGPD):** o link público nunca expõe e-mail, endereço completo ou dados do motorista além do primeiro nome.
- **RNF03 Desempenho:** respostas da API abaixo de 200 ms no p95 para as rotas de leitura, em ambiente local.
- **RNF04 Mobile:** o app do motorista precisa funcionar bem em tela de 360 px e conexão lenta.
- **RNF05 Qualidade:** testes automatizados para regras de negócio e handlers; CI bloqueia merge com teste, lint ou vulnerabilidade conhecida.
- **RNF06 Simplicidade:** rodar o projeto inteiro só com Docker (`docker compose up`).

## Fora de escopo (por enquanto)

- Mapa com GPS do motorista em tempo real, cálculo de frete, rota pelas ruas (a ordem usa distância em linha reta).
- Serviços pagos ou com chave de API (mapas, CEP, geocodificação).
- App nativo (iOS/Android); o motorista usa web mobile.
- Pagamentos, notas fiscais, integração com marketplaces.
- Cadastro aberto de motorista: só a transportadora cria a conta do motorista.
- Vários usuários por transportadora, convites e papéis dentro da transportadora.
- Planos, cobrança e limites por transportadora.

Pedir algo desta lista significa primeiro mudar este documento.

## Decisões sobre perguntas que estavam em aberto

- **Entrega com falha** (`failed`) pode voltar para `in_transit` numa nova tentativa; não vira uma entrega nova. O histórico guarda as duas tentativas.
- **Foto de comprovante** fica fora da v1. Se entrar depois, segue as regras de upload do [SECURITY.md](SECURITY.md).
- **Link público** continua acessível por 30 dias depois de `delivered` ou do último `failed`; depois responde como "não encontrada".
