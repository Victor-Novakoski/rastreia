import { ApiError, request } from './api'

export type Role = 'admin' | 'driver'
export type Session = { token: string; role: Role }

type SessionResponse = { token: string; role: Role; expires_in: number }

// O access token fica só em memória (SECURITY.md #14 e #20); o refresh token
// está num cookie HttpOnly que o JavaScript não lê.
let refreshing: Promise<Session | null> | null = null

/**
 * Troca o cookie por um access token novo. Chamadas simultâneas dividem a
 * mesma requisição: cada refresh token vale uma vez só, e usar o mesmo duas
 * vezes derruba a sessão na API.
 */
export function refreshSession(): Promise<Session | null> {
  refreshing ??= request<SessionResponse>('/auth/refresh', { method: 'POST', withCredentials: true })
    .then(toSession)
    .catch((err: unknown) => {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) return null
      throw err
    })
    .finally(() => {
      refreshing = null
    })
  return refreshing
}

export async function loginRequest(email: string, password: string): Promise<Session> {
  const res = await request<SessionResponse>('/auth/login', {
    method: 'POST',
    body: { email, password },
    withCredentials: true,
  })
  return toSession(res)
}

export function logoutRequest(): Promise<void> {
  return request<void>('/auth/logout', { method: 'POST', withCredentials: true })
}

function toSession(res: SessionResponse): Session {
  return { token: res.token, role: res.role }
}
