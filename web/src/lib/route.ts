import { useQuery } from '@tanstack/react-query'
import { useAuth, type AuthContextValue } from './auth'
import type { Delivery } from './deliveries'

/** Pacote na rota; `position` numera de 1 a N na ordem de entrega (`number` é o da rua). */
export type RoutePackage = Delivery & { position: number }

/** Um endereço da rota; pacotes no mesmo endereço são entregues juntos. */
export type Stop = {
  number: number
  address: string
  latitude: number | null
  longitude: number | null
  packages: RoutePackage[]
}

export type Route = { date: string; total_packages: number; stops: Stop[] }

type Api = AuthContextValue['api']

export const routeKey = ['me', 'route']

export function useRoute() {
  const { api } = useAuth()
  return useQuery({
    queryKey: routeKey,
    queryFn: ({ signal }) => api<Route>('/me/route', { signal }),
    retry: 3,
  })
}

/** Carrega um pacote pelo código ou pelo link lido do QR-code. */
export function addToRoute(api: Api, code: string) {
  return api<Route>('/me/route/deliveries', { method: 'POST', body: { code } })
}

export function removeFromRoute(api: Api, deliveryID: number) {
  return api<Route>(`/me/route/deliveries/${deliveryID}`, { method: 'DELETE' })
}

/** Salva a ordem escolhida: todos os pacotes, uma vez cada. */
export function reorderRoute(api: Api, deliveryIDs: number[]) {
  return api<Route>('/me/route/order', { method: 'PUT', body: { delivery_ids: deliveryIDs } })
}

/** Pede a ordem sugerida, a partir de onde o motorista está quando dá para saber. */
export function optimizeRoute(api: Api, from: { latitude: number; longitude: number } | null) {
  return api<Route>('/me/route/optimize', { method: 'POST', body: from ?? {} })
}

/** Ids dos pacotes na ordem das paradas, para mandar ao reordenar. */
export function packageIDs(stops: Stop[]): number[] {
  return stops.flatMap((s) => s.packages.map((p) => p.id))
}

/** Posição atual do aparelho, ou null se a pessoa não deixar ou demorar. */
export function currentPosition(): Promise<{ latitude: number; longitude: number } | null> {
  return new Promise((resolve) => {
    if (!('geolocation' in navigator)) return resolve(null)
    navigator.geolocation.getCurrentPosition(
      (p) => resolve({ latitude: p.coords.latitude, longitude: p.coords.longitude }),
      () => resolve(null),
      { enableHighAccuracy: true, timeout: 8000, maximumAge: 60_000 },
    )
  })
}
