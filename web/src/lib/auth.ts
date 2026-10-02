import { createContext, useContext } from 'react'
import type { RequestOptions } from './api'
import type { Role, Session } from './session'

export type AuthState =
  | { status: 'loading' }
  | { status: 'anonymous' }
  | { status: 'error' }
  | { status: 'authenticated'; session: Session }

export type AuthContextValue = {
  state: AuthState
  login: (email: string, password: string) => Promise<Role>
  logout: () => Promise<void>
  retry: () => void
  /** Requisição autenticada. Um 401 renova o token uma vez e repete a chamada. */
  api: <T>(path: string, opts?: Omit<RequestOptions, 'token' | 'withCredentials'>) => Promise<T>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth outside AuthProvider')
  return ctx
}

/** Tela inicial de cada papel depois do login. */
export function homeFor(role: Role): string {
  return role === 'admin' ? '/admin/entregas' : '/motorista'
}
