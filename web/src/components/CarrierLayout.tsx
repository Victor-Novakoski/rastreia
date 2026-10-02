import { useQueryClient } from '@tanstack/react-query'
import { NavLink, Outlet } from 'react-router'
import { useAuth } from '../lib/auth'
import { useMe } from '../lib/me'
import type { PanelChange } from '../lib/deliveries'
import { useLive } from '../lib/useLive'
import { LiveBadge } from './LiveBadge'
import { Logo } from './Logo'

const link = ({ isActive }: { isActive: boolean }) =>
  `rounded-md px-3 py-2 font-medium focus-visible:outline-2 focus-visible:outline-white ${isActive ? 'bg-brand-700' : 'hover:bg-brand-700/60'}`

export function CarrierLayout() {
  const { logout } = useAuth()
  const queryClient = useQueryClient()
  // Dados de uma sessão não ficam no cache para a próxima.
  const exit = () => void logout().finally(() => queryClient.clear())
  const live = usePanelLive()
  const me = useMe()
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="bg-brand-800 text-white">
        <nav className="mx-auto flex max-w-6xl flex-wrap items-center gap-2 px-4 py-2" aria-label="Painel da transportadora">
          <span className="mr-4 flex items-center gap-2">
            <Logo className="size-7" />
            <span className="flex flex-col leading-tight">
              <span className="font-bold">Rastreia</span>
              {me.data && <span className="text-sm text-brand-50/80">{me.data.carrier.name}</span>}
            </span>
          </span>
          <NavLink to="/transportadora" end className={link}>
            Visão geral
          </NavLink>
          <NavLink to="/transportadora/entregas" className={link}>
            Entregas
          </NavLink>
          <NavLink to="/transportadora/motoristas" className={link}>
            Motoristas
          </NavLink>
          <span className="ml-auto">{live && <LiveBadge tone="dark" />}</span>
          <button type="button" onClick={exit} className="rounded-md px-3 py-2 font-medium hover:bg-brand-700/60">
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

/** Recarrega listas e detalhes quando outra pessoa muda uma entrega. */
function usePanelLive(): boolean {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()
  return useLive('/live/deliveries', {
    token: accessToken,
    onMessage: (data) => {
      const { delivery_id: id } = data as PanelChange
      void queryClient.invalidateQueries({ queryKey: ['deliveries'] })
      void queryClient.invalidateQueries({ queryKey: ['delivery', id] })
      void queryClient.invalidateQueries({ queryKey: ['events', id] })
    },
    // Durante a queda alguma mudança pode ter passado sem aviso.
    onOpen: (again) => {
      if (again) void queryClient.invalidateQueries()
    },
  })
}
