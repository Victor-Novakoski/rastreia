import { useQueryClient } from '@tanstack/react-query'
import { Link, NavLink, Outlet } from 'react-router'
import { useAuth } from '../lib/auth'
import { useMe } from '../lib/me'

/** Moldura do app do motorista: uma coluna, pensada para o celular. */
export function DriverLayout() {
  const { logout } = useAuth()
  const queryClient = useQueryClient()
  const exit = () => void logout().finally(() => queryClient.clear())
  const me = useMe()
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-10 bg-brand-800 text-white">
        <div className="mx-auto flex max-w-xl items-center justify-between px-4 py-2">
          <Link to="/motorista" className="flex flex-col py-1 leading-tight">
            <span className="text-lg font-bold">Minhas entregas</span>
            {me.data && <span className="text-sm text-brand-50/80">{me.data.carrier.name}</span>}
          </Link>
          <button type="button" onClick={exit} className="min-h-11 rounded-md px-3 font-medium hover:bg-brand-700/60">
            Sair
          </button>
        </div>
        <nav aria-label="Áreas do motorista" className="mx-auto flex max-w-xl px-2">
          <Tab to="/motorista/rota">Rota de hoje</Tab>
          <Tab to="/motorista" end>
            Entregas
          </Tab>
        </nav>
      </header>
      <main className="mx-auto w-full max-w-xl flex-1 px-4 py-4">
        <Outlet />
      </main>
    </div>
  )
}

function Tab({ to, end, children }: { to: string; end?: boolean; children: string }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        `flex min-h-11 flex-1 items-center justify-center border-b-4 font-semibold ${isActive ? 'border-white' : 'border-transparent text-brand-50/80'}`
      }
    >
      {children}
    </NavLink>
  )
}
