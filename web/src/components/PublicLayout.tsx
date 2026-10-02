import type { ReactNode } from 'react'
import { Link } from 'react-router'

/** Moldura das páginas públicas: cabeçalho simples e conteúdo estreito, pensado para celular. */
export function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="bg-brand-800 text-white">
        <div className="mx-auto flex max-w-xl items-center px-4 py-3">
          <Link to="/rastreio" className="text-lg font-bold tracking-tight">
            Rastreia
          </Link>
        </div>
      </header>
      <main className="mx-auto w-full max-w-xl flex-1 px-4 py-6">{children}</main>
    </div>
  )
}
