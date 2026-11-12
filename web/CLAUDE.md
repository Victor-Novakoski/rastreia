# web: front-end

Vite + React + TypeScript + Tailwind, React Router e TanStack Query. Aplicação estática que chama a API pela `VITE_API_URL`. Comandos: `npm test`, `npm run lint` (oxlint, sem avisos), `npm run typecheck`.

## Regras

- **Sessão:** access token só em memória, renovado no 401 por `/auth/refresh`. A renovação passa por uma trava entre abas (`navigator.locks`); duas renovações com o mesmo refresh token derrubam a sessão.
- A sessão é conferida só nas telas que dependem dela. O rastreio público não chama `/auth/refresh`.
- Sair só quando a API confirma; cache limpo ao trocar de conta.
- **CSP** numa `<meta>` injetada no build (`csp.ts`); só libera os hosts de OpenStreetMap, ViaCEP e Nominatim. `frame-ancestors` precisa ir no cabeçalho do host.
- Mapa (Leaflet) e leitor de QR (jsQR, com `BarcodeDetector` antes) são carregados só nas telas que usam; se não baixam, o resto da tela continua.
- Rotas em português (`/transportadora`, `/motorista`, `/rastreio/:código`). Fonte do sistema e ícones SVG próprios.
- Validação igual aos limites da API, com os `fields` do 422 traduzidos embaixo de cada campo. Alvos de toque com 44 px no app do motorista.
- Service worker (`public/sw.js`) só é registrado ao ativar os avisos; não há cache offline.

## Armadilhas conhecidas

- **O navegador guarda a falha de um import dinâmico:** um pedaço (como o mapa) que falhou ao baixar não baixa de novo sem recarregar a página.
- **Arraste por eventos de ponteiro:** quando o React move o elemento no DOM, o navegador solta a captura do ponteiro. O arraste segue o dedo pela janela, não pelo elemento.
- **Na nuvem:** Playwright funciona com `executablePath: '/opt/pw-browsers/chromium'`. Tiles do mapa, ViaCEP e Nominatim não são acessíveis de lá: mocke essas chamadas no Playwright.

## Ver também

Decisões [063](../docs/decisoes/063-csp-do-front-numa-meta-injetada-no.md), [066](../docs/decisoes/066-tanstack-query-no-painel.md), [067](../docs/decisoes/067-access-token-so-em-memoria-renovado-no.md), [081](../docs/decisoes/081-trava-entre-abas-navigator-locks-na-renovacao.md), [096](../docs/decisoes/096-a-sessao-e-conferida-so-nas-telas.md). Design em [docs/DESIGN.md](../docs/DESIGN.md).
