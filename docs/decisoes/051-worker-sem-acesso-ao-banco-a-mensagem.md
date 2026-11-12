# 051. Worker sem acesso ao banco; a mensagem leva nome e e-mail

- **Data:** 2026-10-02
- **Status:** ativa

## Contexto

Menos permissão no worker.

## Decisão

Worker sem acesso ao banco; a mensagem leva nome e e-mail.

Alternativas descartadas: worker buscando a entrega pelo id.

## Consequência

Vale para o e-mail; o push passou a ler o banco (ver "Worker do push lê o banco").
