const baseURL = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(/\/$/, '')

/** Erro no formato da API: { error, fields? } (DESIGN.md). */
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

export async function apiGet<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res: Response
  try {
    res = await fetch(baseURL + path, { ...init, headers: { Accept: 'application/json', ...init.headers } })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'network error')
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string; fields?: Record<string, string> } | null
    throw new ApiError(res.status, body?.error ?? res.statusText, body?.fields)
  }
  return (await res.json()) as T
}
