# PRD — Rastreia

Documento de requisitos do produto. Define **o que** o Rastreia é e **o que não é**. Qualquer funcionalidade nova precisa caber aqui antes de entrar em [TASKS.md](TASKS.md).

## Problema

Transportadoras pequenas e médias acompanham entregas por planilha e WhatsApp. O cliente final não sabe onde está o pedido e liga para perguntar; o motorista não tem um jeito simples de registrar o que aconteceu; a transportadora só descobre um problema quando o cliente reclama.

## Solução

Uma plataforma de rastreio de entregas com três pontas:

1. **Painel da transportadora** (admin): cadastra motoristas e entregas, atribui entregas e acompanha tudo.
2. **App do motorista** (web mobile): vê as próprias entregas e atualiza o status na rua, com poucos toques.
3. **Link público de rastreio** (cliente final): abre pelo código de rastreio, sem login, e vê o status em tempo real.

## Personas

| Persona | Quem é | O que precisa |
| --- | --- | --- |
| Admin | Operador da transportadora | Cadastrar e distribuir entregas rápido; ver o que está atrasado ou falhou |
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

O status só muda por **eventos** registrados pelo motorista (ou admin), nunca por edição direta. Cada evento guarda quem mudou, quando e uma observação opcional, formando o histórico que o cliente vê.

## Requisitos funcionais

### Etapa 1 — Base da API (feito)
- **RF01** Login com e-mail e senha para admin e motorista, devolvendo um token.
- **RF02** Admin cadastra e lista motoristas.
- **RF03** Admin cria, lista (com filtro por status e paginação), consulta e edita entregas.
- **RF04** Toda entrega recebe um código de rastreio aleatório e legível (ex.: `RS7K2M9QXA4P`).

### Etapa 2 — Eventos e rastreio público
- **RF05** Motorista lista apenas as entregas atribuídas a ele.
- **RF06** Motorista (ou admin) registra um evento de status seguindo as transições válidas do ciclo acima.
- **RF07** Consulta pública por código de rastreio, sem login, mostrando status e histórico, **sem dados pessoais** do destinatário (e-mail e endereço completo ficam de fora).

### Etapa 3 — Front-end
- **RF08** Painel admin web.
- **RF09** App web mobile do motorista.
- **RF10** Página pública de rastreio.

### Etapas seguintes
- **RF11** Atualização em tempo real da página de rastreio e do painel (WebSocket).
- **RF12** Notificação por e-mail ao destinatário quando o status muda (fila).

## Requisitos não funcionais

- **RNF01 Segurança:** seguir [SECURITY.md](SECURITY.md). Nenhuma etapa é concluída com item crítico pendente.
- **RNF02 Privacidade (LGPD):** o link público nunca expõe e-mail, endereço completo ou dados do motorista além do primeiro nome.
- **RNF03 Desempenho:** respostas da API abaixo de 200 ms no p95 para as rotas de leitura, em ambiente local.
- **RNF04 Mobile:** o app do motorista precisa funcionar bem em tela de 360 px e conexão lenta.
- **RNF05 Qualidade:** testes automatizados para regras de negócio e handlers; CI bloqueia merge com teste, lint ou vulnerabilidade conhecida.
- **RNF06 Simplicidade:** rodar o projeto inteiro só com Docker (`docker compose up`).

## Fora de escopo (por enquanto)

- Várias transportadoras na mesma instalação (multi-tenant).
- Roteirização, mapa com GPS do motorista em tempo real, cálculo de frete.
- App nativo (iOS/Android); o motorista usa web mobile.
- Pagamentos, notas fiscais, integração com marketplaces.
- Cadastro aberto de usuários: só o admin cria contas.

Pedir algo desta lista significa primeiro mudar este documento.

## Perguntas em aberto

- `failed` pode voltar para `in_transit` (nova tentativa) ou vira uma nova entrega?
- Motorista pode registrar foto como comprovante de entrega? Se sim, entra a parte de upload em [SECURITY.md](SECURITY.md).
- Quanto tempo o link público continua acessível depois de `delivered`?
