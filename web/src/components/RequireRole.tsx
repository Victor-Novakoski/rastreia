import type { ReactNode } from 'react'
import { Navigate } from 'react-router'
import { homeFor, loginFor, useAuth } from '../lib/auth'
import type { Role } from '../lib/session'
import { Loading } from './States'

/** Só mostra o conteúdo para quem está logado com o papel certo. */
export function RequireRole({ role, children }: { role: Role; children: ReactNode }) {
  const { state, retry } = useAuth()
  if (state.status === 'loading') return <Loading label="Verificando sessão" />
  if (state.status === 'error') {
    return (
      <div className="mx-auto max-w-sm px-4 py-10 text-center">
        <p className="font-semibold">Sem conexão com o servidor.</p>
        <button type="button" onClick={retry} className="mt-2 font-semibold text-brand-700 underline underline-offset-2">
          Tentar de novo
        </button>
      </div>
    )
  }
  if (state.status === 'anonymous') return <Navigate to={loginFor(role)} replace />
  if (state.session.role !== role) return <Navigate to={homeFor(state.session.role)} replace />
  return children
}
