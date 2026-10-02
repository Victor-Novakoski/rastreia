import type { ReactNode } from 'react'
import { errorMessage } from '../lib/api'
import { Spinner } from './Spinner'

/** Estados de carregando e erro iguais em todas as telas (DESIGN.md: todo estado tem tela). */
export function Loading({ label = 'Carregando' }: { label?: string }) {
  return (
    <div className="flex justify-center py-10 text-slate-600">
      <Spinner label={label} />
    </div>
  )
}

export function LoadError({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="font-semibold">Não foi possível carregar</p>
      <p className="mt-1 text-slate-600">{errorMessage(error)}</p>
      {onRetry && (
        <button type="button" onClick={onRetry} className="mt-2 font-semibold text-brand-700 underline underline-offset-2">
          Tentar de novo
        </button>
      )}
    </div>
  )
}

export function Empty({ children }: { children: ReactNode }) {
  return <p className="rounded-lg border border-dashed border-slate-300 bg-white p-6 text-center text-slate-600">{children}</p>
}
