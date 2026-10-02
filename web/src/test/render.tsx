import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router'
import { vi } from 'vitest'
import { App } from '../App'
import { AuthProvider } from '../components/AuthProvider'

type Handler = (req: { method: string; path: string; body: unknown; headers: Headers }) => [number, unknown?]

export type Call = { method: string; path: string; body: unknown; headers: Headers; credentials?: RequestCredentials }

/**
 * Troca o fetch por um roteador de mentira: a chave é "MÉTODO /caminho" sem a
 * query string. Rotas sem handler respondem 404. Devolve as chamadas feitas.
 */
export function mockApi(routes: Record<string, Handler>): Call[] {
  const calls: Call[] = []
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
    const url = new URL(String(input))
    const method = init?.method ?? 'GET'
    const headers = new Headers(init?.headers)
    const body = init?.body ? JSON.parse(String(init.body)) : undefined
    calls.push({ method, path: url.pathname + url.search, body, headers, credentials: init?.credentials })
    const handler = routes[`${method} ${url.pathname}`]
    const [status, json] = handler ? handler({ method, path: url.pathname, body, headers }) : [404, { error: 'not found' }]
    return new Response(status === 204 ? null : JSON.stringify(json ?? {}), {
      status,
      headers: { 'Content-Type': 'application/json' },
    })
  })
  return calls
}

/** A aplicação inteira numa rota, como o navegador veria. */
export function renderApp(path: string, extra?: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={client}>
        <AuthProvider>
          <App />
          {extra}
        </AuthProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

export const adminSession = { token: 'access-token', role: 'admin', expires_in: 900 }
