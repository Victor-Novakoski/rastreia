# Design

Convenções de design da API e das interfaces. O objetivo é que tudo pareça feito pela mesma pessoa.

## API

### Recursos e rotas
- REST com JSON. Recursos no plural: `/deliveries`, `/drivers`.
- Rotas só do motorista ficam sob `/me/...` (ex.: `/me/deliveries`, `/me/route`), sempre filtradas pelo usuário do token. Rotas por id (`/deliveries/{id}` e os eventos) servem a transportadora e o motorista e conferem o dono.
- Rotas públicas ficam sob `/public/...` (ex.: `/public/tracking/{code}`) e nunca recebem id interno, só o código de rastreio.
- Mudança de status é um recurso próprio: `POST /deliveries/{id}/events`. Não existe `PATCH` de status.

### Formato
- Campos em `snake_case`.
- Datas em RFC 3339 (`2026-10-01T20:00:00Z`), no fuso do servidor, que é UTC no Docker e em produção.
- Ids numéricos (`int64`) só nas rotas autenticadas. O código de rastreio é o identificador público.
- Nenhum campo some do JSON. Sem valor, ids, números, datas, o CNPJ e a observação do evento vêm como `null`; as partes opcionais do endereço (`complement`, `address_reference`) vêm como string vazia.
- Corpo JSON com limite de 1 MB e campos desconhecidos rejeitados.

### Erros
Sempre o mesmo formato, inclusive para rota que não existe e método errado:

```json
{ "error": "mensagem curta" }
```

Validação (422) traz os campos:

```json
{ "error": "invalid input", "fields": { "recipient_email": "must be a valid e-mail" } }
```

| Status | Quando |
| --- | --- |
| 400 | JSON inválido, corpo vazio, campo desconhecido, id malformado |
| 401 | Sem token, token inválido ou login errado |
| 403 | Autenticado, mas sem permissão; ou `Origin` fora da lista nas rotas de sessão |
| 404 | Não existe (ou não pertence a quem pede, para não revelar que existe); rota inexistente |
| 405 | Método que a rota não aceita (com o cabeçalho `Allow`) |
| 409 | Conflito (ex.: e-mail já usado, transição de status inválida, entrega já anonimizada) |
| 422 | Dados inválidos, com `fields` |
| 429 | Limite de requisições excedido, com `Retry-After` |
| 499 | O cliente desistiu antes da resposta (só aparece no log) |
| 500 | Erro inesperado, mensagem genérica |
| 503 | Dependência fora do ar: banco ou Redis no `/health`, Redis no login |
| 504 | A requisição passou de 15 s |

### Paginação
- `?page=1&size=20`. `size` acima de 100 ou inválido volta para 20; `page` inválida vira 1, e acima de 10000 responde 422.
- Filtros somam: `status`, `driver` (`none` ou o id do motorista) e `q` (busca). Valor inválido responde 422.
- Ordenação padrão: mais recentes primeiro. Em `/me/deliveries`, as que faltam fazer vêm antes das entregues.

## Interfaces

Quatro superfícies, uma identidade visual:

| Superfície | Dispositivo principal | Prioridade |
| --- | --- | --- |
| Página inicial | Desktop e celular | Entender o produto em 5 segundos e achar a própria porta: transportadora, motorista ou rastreio |
| Painel da transportadora | Desktop | Densidade de informação: visão geral, tabela com busca, filtros por status e por motorista e status visível |
| App do motorista | Celular, uma mão, na rua | Botões grandes, poucos toques, funciona com sinal ruim |
| Rastreio público | Celular, link vindo de e-mail | Carregar rápido; status e linha do tempo entendidos em 3 segundos |

### Princípios
- **Mobile first** para motorista e rastreio público; área de toque mínima de 44 px.
- **Todo estado tem tela:** carregando, vazio, erro e sucesso. Nada de tela branca.
- **Bloquear durante envio:** botão desabilitado e com indicador enquanto a requisição está em andamento, para evitar envio duplo.
- **Feedback de erro no campo:** os `fields` do 422 aparecem embaixo do campo correspondente.
- **Status sempre com cor + texto + ícone**, nunca só cor (acessibilidade).
- **Textos em português do Brasil**, datas em `dd/mm/aaaa HH:mm` no fuso do usuário.
- Contraste mínimo WCAG AA; navegação por teclado no painel da transportadora.

### Status na interface

| Status | Rótulo | Cor (semântica) |
| --- | --- | --- |
| `pending` | Aguardando coleta | neutra |
| `picked_up` | Coletado | informação |
| `in_transit` | Em rota | informação (destaque) |
| `delivered` | Entregue | sucesso |
| `failed` | Não entregue | erro |

### Paleta
Tokens em `web/src/index.css` (`@theme` do Tailwind). Componentes usam o nome semântico, nunca a cor crua.

| Token | Cor | Uso |
| --- | --- | --- |
| `brand-700` / `brand-800` | `#4338ca` / `#3730a3` | Botão principal, links, cabeçalho |
| `neutral-fg` / `neutral-bg` | `#475569` / `#f1f5f9` | `pending` |
| `info-fg` / `info-bg` | `#0369a1` / `#e0f2fe` | `picked_up` |
| `highlight-fg` / `highlight-bg` | `#1d4ed8` / `#dbeafe` | `in_transit` |
| `success-fg` / `success-bg` | `#15803d` / `#dcfce7` | `delivered` |
| `danger-fg` / `danger-bg` | `#b91c1c` / `#fee2e2` | `failed`, erros de formulário |

Fundo `slate-50`, texto `slate-900` e texto secundário `slate-600`. Todos os pares de texto e fundo passam no contraste AA.

### Tipografia
Fonte do sistema (`system-ui`): nada para baixar, o que ajuda o rastreio público a abrir rápido em rede ruim. Código de rastreio em fonte monoespaçada, com espaçamento entre letras.

### Componentes
- `StatusBadge`: status com ícone, texto e cor; tamanho `lg` no topo do rastreio.
- `Button`: altura mínima de 44 px; com `loading`, fica desabilitado e mostra o indicador.
- `Spinner`: indicador de carregamento com texto para leitor de tela.
- `Logo`: marca (pino com caixa) em SVG, sempre ao lado do nome.
- `SiteHeader`: cabeçalho das páginas abertas, com Rastrear e, na página inicial, Sou motorista e Entrar.
- `PublicLayout`: cabeçalho e coluna estreita das páginas públicas.
- `AuthCard`: cartão central das telas de entrar e cadastrar, com links para as outras áreas embaixo.
- `CarrierLayout`: barra de navegação do painel (Visão geral, Entregas, Motoristas, Sair), com o nome da transportadora, e conteúdo até 1152 px.
- `TextField`, `TextArea`, `SelectField`: rótulo, controle com 44 px de altura e erro embaixo, ligado por `aria-describedby`.
- `Alert`: erro (`role="alert"`) ou sucesso (`role="status"`) no topo do formulário.
- `Loading`, `LoadError`, `Empty`: os estados de carregando, erro (com "Tentar de novo") e vazio.
- `DriverLayout`: cabeçalho fixo com o nome da transportadora, as abas "Rota de hoje" e "Entregas" e uma coluna para o celular. No app do motorista, os botões de status têm 56 px de altura e dizem a ação ("Saí para entrega", "Entreguei"); "Não consegui entregar" é contornado em vermelho e abre o campo do motivo.
- `DeliveryForm`: entrega com CEP (o ViaCEP preenche rua, bairro, cidade e UF), telefone com máscara e pino no mapa. Mudar o endereço tira o pino antigo.
- `PinMap` e `StopsMap` (`Map.tsx`, carregados sob demanda por `LazyMap.tsx`): pino arrastável do endereço, que também dá para pôr pelo teclado, e as paradas numeradas da rota. Se o mapa não baixa, um aviso fica no lugar dele e o resto da tela continua.
- `QrScanner`: leitor do QR-code da etiqueta (`BarcodeDetector`, com jsQR de reserva). Sem câmera ou sem o leitor, o motorista digita o código.
- `EventForm`: troca de status no painel, só com as transições que a API aceita.
- `PushToggle`: liga e desliga o aviso no celular na página de rastreio, e explica quando o navegador bloqueou ou quando o iPhone precisa do site na tela de início.
- `LiveBadge`: selo "Ao vivo" enquanto o WebSocket está aberto.
- `RequireRole`: só mostra a área a quem tem o papel. Sem sessão, leva ao login daquela área; com o outro papel, leva à área certa.
- `ErrorBoundary` e `Crashed`: erro inesperado mostra uma tela com "Recarregar", nunca uma tela branca.
- Etiqueta (`LabelPage`): 10 x 15 cm, pronta para imprimir, com o QR-code do link de rastreio. "Imprimir" só aparece quando o QR-code e o nome da transportadora estão prontos.

Ícones são SVG próprios em `StatusIcon`, sem biblioteca.
