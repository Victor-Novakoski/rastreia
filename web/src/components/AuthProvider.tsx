import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { ApiError, request, type RequestOptions } from '../lib/api'
import { AuthContext, type AuthState } from '../lib/auth'
import { loginRequest, logoutRequest, refreshSession, signUpRequest, type Session, type SignUpInput } from '../lib/session'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: 'loading' })
  const [attempt, setAttempt] = useState(0)
  const sessionRef = useRef<Session | null>(null)

  const apply = useCallback((session: Session | null) => {
    sessionRef.current = session
    setState(session ? { status: 'authenticated', session } : { status: 'anonymous' })
  }, [])

  // Ao abrir a página, o cookie diz se ainda existe uma sessão.
  useEffect(() => {
    let active = true
    refreshSession()
      .then((s) => active && apply(s))
      .catch(() => active && setState({ status: 'error' }))
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

  const logout = useCallback(async () => {
    try {
      await logoutRequest()
    } finally {
      apply(null)
    }
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

