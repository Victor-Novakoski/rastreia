import { useQueryClient } from '@tanstack/react-query'
import { NavLink, Outlet } from 'react-router'
import { useAuth } from '../lib/auth'

const link = ({ isActive }: { isActive: boolean }) =>
  `rounded-md px-3 py-2 font-medium focus-visible:outline-2 focus-visible:outline-white ${isActive ? 'bg-brand-700' : 'hover:bg-brand-700/60'}`

export function AdminLayout() {
  const { logout } = useAuth()
  const queryClient = useQueryClient()
  // Dados de uma sessão não ficam no cache para a próxima.
  const exit = () => void logout().finally(() => queryClient.clear())
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="bg-brand-800 text-white">
        <nav className="mx-auto flex max-w-6xl flex-wrap items-center gap-2 px-4 py-2" aria-label="Painel">
          <span className="mr-4 text-lg font-bold">Rastreia</span>
          <NavLink to="/admin/entregas" className={link}>
            Entregas
          </NavLink>
          <NavLink to="/admin/motoristas" className={link}>
            Motoristas
          </NavLink>
          <button type="button" onClick={exit} className="ml-auto rounded-md px-3 py-2 font-medium hover:bg-brand-700/60">
            Sair
          </button>
        </nav>
      </header>
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
