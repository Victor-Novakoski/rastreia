import { lazy, Suspense, type ComponentProps, type ReactNode } from 'react'
import { ErrorBoundary } from './ErrorBoundary'
import type { PinMap as PinMapView, StopsMap as StopsMapView } from './Map'

// O Leaflet só é baixado nas telas com mapa, não no rastreio público nem no login.
const LazyPinMap = lazy(() => import('./Map').then((m) => ({ default: m.PinMap })))
const LazyStopsMap = lazy(() => import('./Map').then((m) => ({ default: m.StopsMap })))

const placeholder = <div className="h-64 w-full rounded-lg border border-slate-300 bg-slate-100" />

/**
 * Sem sinal, ou com a aba aberta desde antes de uma atualização do site, o
 * mapa pode não baixar. O aviso fica no lugar dele e o resto da tela segue
 * funcionando. O navegador guarda a falha do download, então tentar de novo
 * na mesma página não adianta: só recarregar traz o mapa.
 */
function MapBoundary({ note, children }: { note: string; children: ReactNode }) {
  return (
    <ErrorBoundary
      fallback={
        <div
          role="alert"
          className="flex min-h-64 w-full flex-col items-center justify-center gap-2 rounded-lg border border-slate-300 bg-slate-50 p-4 text-center"
        >
          <p className="font-medium">Não foi possível carregar o mapa.</p>
          <p className="text-sm text-slate-600">{note}</p>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="min-h-11 px-3 font-semibold text-brand-700 underline underline-offset-2"
          >
            Recarregar a página
          </button>
        </div>
      }
    >
      <Suspense fallback={placeholder}>{children}</Suspense>
    </ErrorBoundary>
  )
}

export function PinMap(props: ComponentProps<typeof PinMapView>) {
  return (
    <MapBoundary note="Dá para salvar a entrega sem o pino e marcar depois. Recarregar apaga o que ainda não foi salvo.">
      <LazyPinMap {...props} />
    </MapBoundary>
  )
}

export function StopsMap(props: ComponentProps<typeof StopsMapView>) {
  return (
    <MapBoundary note="A lista de paradas continua funcionando.">
      <LazyStopsMap {...props} />
    </MapBoundary>
  )
}
