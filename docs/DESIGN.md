# Design

Como o produto parece e se comporta para quem usa.

## Princípios

- **Um fluxo, sem distração:** criar entrega, atualizar status, acompanhar. Cada tela serve a um desses passos.
- **Motorista primeiro no celular:** botões grandes, uma mão, funciona com sinal fraco.
- **Status sempre visível:** a mesma cor e o mesmo nome do status em todas as telas.
- **Acessível:** contraste AA, foco visível, tudo usável pelo teclado, status nunca só por cor (sempre com texto).

## Telas

### 1. Login (admin e motorista)

E-mail, senha e botão Entrar. Erro genérico "E-mail ou senha inválidos". Botão desabilitado com carregando durante o envio. Depois do login, admin vai para Entregas e motorista para Minhas entregas.

### 2. Entregas (admin, desktop)

- Tabela: código, destinatário, motorista, status, atualizado em.
- Filtro por status e paginação.
- Botão "Nova entrega" abre um formulário lateral: destinatário, e-mail, endereço, motorista (opcional).
- Clique na linha abre o detalhe: dados editáveis, histórico de eventos e o link público para copiar.

### 3. Motoristas (admin)

Lista e formulário de cadastro: nome, e-mail, senha inicial.

### 4. Minhas entregas (motorista, celular)

- Cartões com destinatário, endereço e status.
- Cada cartão tem o próximo passo como botão principal ("Coletei", "Saí para entrega", "Entreguei") e "Não consegui entregar" como ação secundária, que pede uma observação.
- Confirmação antes de marcar como entregue ou falhou.

### 5. Rastreio público (cliente, celular e desktop)

- Código de rastreio no topo, status atual em destaque e uma linha do tempo com os eventos.
- Atualiza sozinha (WebSocket), com um indicador discreto de "ao vivo".
- Sem login e sem dados pessoais além do primeiro nome.
- Código inválido mostra "Entrega não encontrada", sem diferenciar de código inexistente.

## Status

| Status | Texto | Cor (Tailwind) |
| --- | --- | --- |
| pending | Aguardando coleta | slate |
| picked_up | Coletado | blue |
| in_transit | Em rota | amber |
| delivered | Entregue | green |
| failed | Não entregue | red |

## Visual

- **Tipografia:** Inter, tamanhos da escala do Tailwind; números da tabela com `tabular-nums`.
- **Cores:** fundo neutro (slate), uma cor de marca (indigo) para ações principais, cores de status só para status.
- **Componentes:** botão (primário, secundário, perigo), input com mensagem de erro abaixo, badge de status, tabela, cartão, toast, modal de confirmação.
- **Estados:** toda tela tem carregando (skeleton), vazio (com ação para começar) e erro (com tentar de novo).
- **Tema escuro:** suportado desde o início, seguindo o sistema.

## Textos

Português, frases curtas, sem jargão técnico. Erros dizem o que fazer: "Informe um e-mail válido", não "invalid input".
