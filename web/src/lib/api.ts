export const baseURL = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(/\/$/, '')

/** Erro no formato da API: { error, fields? } (DESIGN.md). Status 0 = sem conexão. */
export class ApiError extends Error {
  readonly status: number
  readonly fields: Record<string, string>

  constructor(status: number, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fields = fields
  }
}

export type RequestOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  body?: unknown
  token?: string
  headers?: Record<string, string>
  signal?: AbortSignal
  /** Manda e recebe o cookie de sessão; só as rotas /auth/... precisam. */
  withCredentials?: boolean
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json', ...opts.headers }
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`

  let res: Response
  try {
    res = await fetch(baseURL + path, {
      method: opts.method ?? 'GET',
      headers,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
      signal: opts.signal,
      credentials: opts.withCredentials ? 'include' : 'omit',
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'network error')
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string; fields?: Record<string, string> } | null
    throw new ApiError(res.status, body?.error ?? res.statusText, body?.fields)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export function apiGet<T>(path: string, init: { signal?: AbortSignal } = {}): Promise<T> {
  return request<T>(path, init)
}

/** Mensagem curta em português para erros que não são de campo. */
export function errorMessage(err: unknown): string {
  if (!(err instanceof ApiError)) return 'Algo deu errado. Tente de novo.'
  switch (err.status) {
    case 0:
      return 'Sem conexão com o servidor.'
    case 403:
      return 'Você não tem permissão para isso.'
    case 404:
      return 'Não encontrado.'
    case 409:
      return 'Conflito com o estado atual. Recarregue e tente de novo.'
    case 422:
      return 'Confira os campos destacados.'
    case 429:
      return 'Muitas tentativas seguidas. Espere um pouco.'
    default:
      return 'Algo deu errado do nosso lado.'
  }
}
