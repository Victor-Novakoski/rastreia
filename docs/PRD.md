# PRD: Rastreia

## Problema

Pequenas transportadoras controlam entregas por planilha e WhatsApp. O cliente final não sabe onde está o pedido e liga para perguntar, e o motorista não tem um jeito simples de registrar o que aconteceu em cada parada.

## Solução

Um sistema web com três lados:

- **Painel da transportadora (admin):** cadastra motoristas e entregas e acompanha tudo em uma lista.
- **Tela do motorista (celular):** vê as entregas atribuídas a ele e muda o status de cada uma.
- **Página pública de rastreio:** o cliente abre um link com o código de rastreio e vê o status mudar em tempo real, sem login.

## Usuários

| Papel | O que faz | Como entra |
| --- | --- | --- |
| Admin | Cadastra motoristas e entregas, edita e acompanha | E-mail e senha |
| Motorista | Atualiza o status das entregas dele | E-mail e senha |
| Cliente final | Acompanha uma entrega | Link com código de rastreio, sem login |

## Escopo da v1

1. Login com e-mail e senha para admin e motorista.
2. Admin cadastra e lista motoristas.
3. Admin cria, lista, filtra por status, vê e edita entregas. Cada entrega ganha um código de rastreio único.
4. Motorista vê só as entregas atribuídas a ele e registra mudanças de status: coletado, em rota, entregue ou falhou, com observação opcional.
5. Cada mudança de status vira um evento no histórico da entrega.
6. Página pública mostra status atual e histórico pelo código de rastreio, atualizando sozinha via WebSocket.
7. O destinatário recebe e-mail a cada mudança de status, enviado por uma fila, sem travar a requisição.

## Fora da v1

Multi-empresa, mapa com GPS, app mobile nativo, pagamento e frete, relatórios, WhatsApp, Kubernetes.

## Requisitos não funcionais

- **Segurança:** seguir o [SECURITY.md](SECURITY.md).
- **Testes:** toda regra de negócio tem teste unitário; rotas têm teste de integração com Postgres real.
- **Desempenho:** listagens paginadas; respostas comuns abaixo de 200 ms localmente.
- **Operação:** sobe com um comando (`docker compose up`); deploy automático pela CI.
- **Documentação:** OpenAPI sempre atualizada com as rotas.

## Critérios de sucesso

- Um recrutador abre a demo, entra com o usuário de teste, cria uma entrega e vê o status mudar na página pública em menos de 2 minutos.
- CI verde em todo push, com testes, lint e verificação de vulnerabilidades.
- Projeto no ar na AWS com HTTPS.
