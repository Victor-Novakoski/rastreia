import { useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { ApiError, request, type RequestOptions } from '../lib/api'
import { AuthContext, type AuthState } from '../lib/auth'
import {
  loginRequest,
  logoutRequest,
  refreshSession,
  sameUser,
  signUpRequest,
  type Session,
  type SignUpInput,
} from '../lib/session'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: 'loading' })
  const [attempt, setAttempt] = useState(0)
  const sessionRef = useRef<Session | null>(null)
  const queryClient = useQueryClient()

  const apply = useCallback(
    (session: Session | null) => {
      const before = sessionRef.current
      sessionRef.current = session
      // Os dados de uma conta não ficam no cache para a próxima: nem depois
      // de sair, nem quando a sessão vence ou vira a de outra conta (alguém
      // entrou com outro login em outra aba).
      if (before && !session) queryClient.clear()
      else if (before && session && !sameUser(before, session)) void queryClient.resetQueries()
      setState(session ? { status: 'authenticated', session } : { status: 'anonymous' })
    },
    [queryClient],
  )

  // Ao abrir a página, o cookie diz se ainda existe uma sessão.
  useEffect(() => {
    let active = true
    refreshSession()
      .then((s) => active && apply(s))
      .catch((error: unknown) => active && setState({ status: 'error', error }))
    return () => {
      active = false
    }
  }, [apply, attempt])

  const login = useCallback(
    async (email: string, password: string) => {
      const session = await loginRequest(email, password)
      apply(session)
      return session.role
    },
    [apply],
  )

  const signUp = useCallback(
    async (input: SignUpInput) => {
      const session = await signUpRequest(input)
      apply(session)
      return session.role
    },
    [apply],
  )

  // Se a API não confirmar, a sessão continua: num celular dividido, a
  // próxima pessoa entraria com a conta de quem achou que tinha saído.
  const logout = useCallback(async () => {
    await logoutRequest()
    apply(null)
  }, [apply])

  const api = useCallback(
    async <T,>(path: string, opts: Omit<RequestOptions, 'token' | 'withCredentials'> = {}): Promise<T> => {
      const token = sessionRef.current?.token
      try {
        return await request<T>(path, { ...opts, token })
      } catch (err) {
        if (!(err instanceof ApiError) || err.status !== 401) throw err
        const session = await refreshSession()
        apply(session)
        if (!session) throw err
        return request<T>(path, { ...opts, token: session.token })
      }
    },
    [apply],
  )

  const accessToken = useCallback(
    async (renew: boolean) => {
      if (!renew) return sessionRef.current?.token ?? null
      const session = await refreshSession()
      apply(session)
      return session?.token ?? null
    },
    [apply],
  )

  const value = useMemo(
    () => ({
      state,
      login,
      signUp,
      logout,
      api,
      accessToken,
      retry: () => {
        setState({ status: 'loading' })
        setAttempt((n) => n + 1)
      },
    }),
    [state, login, signUp, logout, api, accessToken],
  )
  return <AuthContext value={value}>{children}</AuthContext>
}

