# 097. Deploy sem custo de hospedagem

- **Data:** 2026-10-09
- **Status:** ativa

## Contexto

O plano de deploy (etapa 7 do antigo TASKS.md e itens 1, 16, 20 e 25 do SECURITY) assumia a AWS: Secrets Manager/SSM, CDN, sub-rede privada. O projeto é de portfólio e não deve ter custo mensal.

## Decisão

Publicar só em planos gratuitos. O provedor ainda não foi escolhido; os requisitos de segurança do deploy continuam (HTTPS, banco sem acesso de fora, segredos fora de arquivo, `frame-ancestors` no cabeçalho do front), adaptados ao que o provedor oferecer.

Alternativas descartadas: AWS, que cobra pelo banco, pelo load balancer e pelos segredos.

## Consequência

Os docs deixam de citar serviços da AWS. As tarefas do deploy viram issues sem provedor fixo, e a escolha do provedor vira uma decisão nova quando for feita.
