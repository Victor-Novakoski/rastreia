import { Link, NavLink } from 'react-router'
import { Logo } from './Logo'

const link = ({ isActive }: { isActive: boolean }) =>
  `rounded-md px-3 py-2 font-medium hover:bg-brand-700/60 focus-visible:outline-2 focus-visible:outline-white ${isActive ? 'bg-brand-700' : ''}`

/** Cabeçalho das páginas abertas: marca, rastreio e as entradas de cada público. */
export function SiteHeader({ wide = false }: { wide?: boolean }) {
  return (
    <header className="bg-brand-800 text-white">
      <nav
        className={`mx-auto flex flex-wrap items-center gap-1 px-4 py-2 ${wide ? 'max-w-6xl' : 'max-w-xl'}`}
        aria-label="Site"
      >
        <Link to="/" className="mr-auto flex items-center gap-2 py-1 text-lg font-bold tracking-tight">
          <Logo className="size-7" />
          Rastreia
        </Link>
        <NavLink to="/rastreio" className={link}>
          Rastrear
        </NavLink>
        {wide && (
          <>
            <NavLink to="/motorista/entrar" className={link}>
              Sou motorista
            </NavLink>
            <NavLink to="/transportadora/entrar" className={link}>
              Entrar
            </NavLink>
          </>
        )}
      </nav>
    </header>
  )
}
