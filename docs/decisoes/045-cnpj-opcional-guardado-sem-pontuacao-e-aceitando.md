# 045. CNPJ opcional, guardado sem pontuação e aceitando o formato alfanumérico

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

A Receita passou a emitir CNPJ com letras em julho de 2026; o dígito verificador usa o código ASCII menos 48, que dá o mesmo resultado para os números.

## Decisão

CNPJ opcional, guardado sem pontuação e aceitando o formato alfanumérico.

Alternativas descartadas: CNPJ obrigatório, que atrapalha quem só quer testar.

## Consequência

Não registrada além do contexto.
