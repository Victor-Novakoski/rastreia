# 029. Coordenadas vêm do front, não de geocodificação na API

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

O formulário mostra o pino no mapa e a transportadora arrasta se cair errado; a API não chama serviço externo e os testes não dependem de rede.

## Decisão

Coordenadas vêm do front, não de geocodificação na API.

Alternativas descartadas: Nominatim na API ao criar, que exige fila para respeitar 1 req/s e não deixa corrigir.

## Consequência

Não registrada além do contexto.
