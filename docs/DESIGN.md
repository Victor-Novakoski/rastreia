# Design

Convenções de design da API e das interfaces. O objetivo é que tudo pareça feito pela mesma pessoa.

## API

### Recursos e rotas
- REST com JSON. Recursos no plural: `/deliveries`, `/drivers`.
- Rotas do motorista ficam sob `/me/...` (ex.: `/me/deliveries`), sempre filtradas pelo usuário do token.
- Rotas públicas ficam sob `/public/...` (ex.: `/public/tracking/{code}`) e nunca recebem id interno, só o código de rastreio.
- Mudança de status é um recurso próprio: `POST /deliveries/{id}/events`. Não existe `PATCH` de status.

### Formato
- Campos em `snake_case`.
- Datas em RFC 3339, em UTC (`2026-10-01T20:00:00Z`).
- Ids numéricos (`int64`) só nas rotas autenticadas. O código de rastreio é o identificador público.
- Campos opcionais ausentes vêm como `null`, não somem do JSON.
- Corpo JSON com limite de 1 MB e campos desconhecidos rejeitados.

### Erros
Sempre o mesmo formato:

```json
{ "error": "mensagem curta" }
```

Validação (422) traz os campos:

```json
{ "error": "invalid input", "fields": { "recipient_email": "must be a valid e-mail" } }
```

| Status | Quando |
| --- | --- |
| 400 | JSON inválido, id malformado |
| 401 | Sem token, token inválido ou login errado |
| 403 | Autenticado, mas sem permissão |
| 404 | Não existe (ou não pertence a quem pede, para não revelar que existe) |
| 409 | Conflito (ex.: e-mail já usado, transição de status inválida) |
| 422 | Dados inválidos, com `fields` |
| 429 | Limite de requisições excedido |
| 500 | Erro inesperado, mensagem genérica |

### Paginação
- `?page=1&size=20`. `size` máximo 100; valores inválidos caem no padrão.
- Ordenação padrão: mais recentes primeiro.

## Interfaces (etapa 3)

Três superfícies, uma identidade visual:

| Superfície | Dispositivo principal | Prioridade |
| --- | --- | --- |
| Painel admin | Desktop | Densidade de informação: tabela com filtros, busca e status visível |
| App do motorista | Celular, uma mão, na rua | Botões grandes, poucos toques, funciona com sinal ruim |
| Rastreio público | Celular, link vindo de e-mail | Carregar rápido; status e linha do tempo entendidos em 3 segundos |

### Princípios
- **Mobile first** para motorista e rastreio público; área de toque mínima de 44 px.
- **Todo estado tem tela:** carregando, vazio, erro e sucesso. Nada de tela branca.
- **Bloquear durante envio:** botão desabilitado e com indicador enquanto a requisição está em andamento, para evitar envio duplo.
- **Feedback de erro no campo:** os `fields` do 422 aparecem embaixo do campo correspondente.
- **Status sempre com cor + texto + ícone**, nunca só cor (acessibilidade).
- **Textos em português do Brasil**, datas em `dd/mm/aaaa HH:mm` no fuso do usuário.
- Contraste mínimo WCAG AA; navegação por teclado no painel admin.

### Status na interface

| Status | Rótulo | Cor (semântica) |
| --- | --- | --- |
| `pending` | Aguardando coleta | neutra |
| `picked_up` | Coletado | informação |
| `in_transit` | Em rota | informação (destaque) |
| `delivered` | Entregue | sucesso |
| `failed` | Não entregue | erro |

Paleta, tipografia e componentes concretos serão definidos no início da etapa 3 e registrados aqui.
