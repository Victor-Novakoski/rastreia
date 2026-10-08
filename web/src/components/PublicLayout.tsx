import type { ReactNode } from 'react'
import { SiteHeader } from './SiteHeader'

/** Moldura das páginas públicas: cabeçalho simples e conteúdo estreito, pensado para celular. */
export function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <SiteHeader />
      <main className="mx-auto w-full max-w-xl flex-1 px-4 py-6">{children}</main>
    </div>
  )
}
