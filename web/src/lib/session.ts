import { ApiError, request } from './api'

export type Role = 'carrier' | 'driver'
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
  refreshing ??= oneTabAtATime(() =>
    request<SessionResponse>('/auth/refresh', { method: 'POST', withCredentials: true }),
  )
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

/**
 * Abas abertas juntas (vários links abertos de uma vez, o navegador
 * restaurando a janela) renovariam com o mesmo cookie, e a API entende o
 * segundo uso como roubo e derruba a sessão de todas. A trava do navegador
 * faz uma aba esperar a outra: quando chega a vez, o cookie já é o novo.
 */
function oneTabAtATime<T>(renew: () => Promise<T>): Promise<T> {
  return navigator.locks ? navigator.locks.request('rastreia-refresh', renew) : renew()
}

export async function loginRequest(email: string, password: string): Promise<Session> {
  const res = await request<SessionResponse>('/auth/login', {
    method: 'POST',
    body: { email, password },
    withCredentials: true,
  })
  return toSession(res)
}

export type SignUpInput = {
  carrier_name: string
  document?: string
  name: string
  email: string
  password: string
}

/** Cria a transportadora e já devolve a sessão de quem a cadastrou, como o login. */
export async function signUpRequest(input: SignUpInput): Promise<Session> {
  const res = await request<SessionResponse>('/auth/signup', { method: 'POST', body: input, withCredentials: true })
  return toSession(res)
}

export function logoutRequest(): Promise<void> {
  return request<void>('/auth/logout', { method: 'POST', withCredentials: true })
}

/** As duas sessões são da mesma conta? Compara o dono (sub) dos tokens. */
export function sameUser(a: Session, b: Session): boolean {
  return a.role === b.role && tokenSubject(a.token) === tokenSubject(b.token)
}

function tokenSubject(token: string): string | null {
  try {
    const payload = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
    const { sub } = JSON.parse(atob(payload)) as { sub?: unknown }
    return typeof sub === 'string' ? sub : null
  } catch {
    return null
  }
}

function toSession(res: SessionResponse): Session {
  return { token: res.token, role: res.role }
}
