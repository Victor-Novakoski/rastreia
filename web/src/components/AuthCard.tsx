import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { Logo } from './Logo'

/** Moldura das telas de entrar e cadastrar: cartão central com a marca e links para as outras áreas. */
export function AuthCard({
  eyebrow,
  title,
  subtitle,
  children,
  footer,
}: {
  eyebrow: string
  title: string
  subtitle?: ReactNode
  children: ReactNode
  footer?: ReactNode
}) {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-6 px-4 py-10">
      <Link to="/" className="flex items-center gap-2 text-lg font-bold text-brand-800">
        <Logo className="size-8" />
        Rastreia
      </Link>
      <main className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-semibold tracking-wide text-brand-700 uppercase">{eyebrow}</p>
        <h1 className="mt-1 text-2xl font-bold">{title}</h1>
        {subtitle && <p className="mt-1 text-slate-600">{subtitle}</p>}
        <div className="mt-6">{children}</div>
      </main>
      {footer && <div className="flex flex-col items-center gap-1 text-center text-sm text-slate-600">{footer}</div>}
    </div>
  )
}

export function AuthLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link to={to} className="font-semibold text-brand-700 underline-offset-2 hover:underline">
      {children}
    </Link>
  )
}
