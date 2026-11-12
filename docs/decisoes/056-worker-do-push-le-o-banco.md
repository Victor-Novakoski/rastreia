# 056. Worker do push lê o banco

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

As inscrições mudam a qualquer hora; mandar todas dentro da mensagem do evento deixaria a fila com dados velhos.

## Decisão

Worker do push lê o banco.

## Consequência

O e-mail continua sem banco.
