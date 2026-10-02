import { lazy, Suspense, type ComponentProps } from 'react'

// O Leaflet só é baixado nas telas com mapa, não no rastreio público nem no login.
const LazyPinMap = lazy(() => import('./Map').then((m) => ({ default: m.PinMap })))
const LazyStopsMap = lazy(() => import('./Map').then((m) => ({ default: m.StopsMap })))

const placeholder = <div className="h-64 w-full rounded-lg border border-slate-300 bg-slate-100" />

export function PinMap(props: ComponentProps<typeof LazyPinMap>) {
  return (
    <Suspense fallback={placeholder}>
      <LazyPinMap {...props} />
    </Suspense>
  )
}

export function StopsMap(props: ComponentProps<typeof LazyStopsMap>) {
  return (
    <Suspense fallback={placeholder}>
      <LazyStopsMap {...props} />
    </Suspense>
  )
}
