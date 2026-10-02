import type { ButtonHTMLAttributes } from 'react'
import { Spinner } from './Spinner'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean; variant?: 'primary' | 'danger' }

const variants = {
  primary: 'bg-brand-700 hover:bg-brand-800 focus-visible:outline-brand-600',
  danger: 'bg-danger-fg hover:bg-red-800 focus-visible:outline-danger-fg',
}

/** Botão principal. Com `loading`, fica desabilitado e mostra o indicador (SECURITY.md #9). */
export function Button({ loading = false, variant = 'primary', disabled, children, className = '', ...rest }: Props) {
  return (
    <button
      {...rest}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={`inline-flex min-h-11 items-center justify-center gap-2 rounded-lg px-4 font-semibold text-white focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-60 ${variants[variant]} ${className}`}
    >
      {loading && <Spinner label="Enviando" />}
      {children}
    </button>
  )
}
