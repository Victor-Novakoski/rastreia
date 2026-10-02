import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { useEffect, useRef } from 'react'
import type { LatLng } from '../lib/address'

// Mapa do OpenStreetMap com Leaflet: gratuito e sem chave. Os marcadores são
// HTML (divIcon), sem imagem, e seguem as cores do DESIGN.md.

const tiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
const attribution = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'

/** Centro do Brasil, para quando ainda não há ponto. */
const brazil: L.LatLngTuple = [-14.2, -51.9]

function useLeaflet(container: React.RefObject<HTMLDivElement | null>) {
  const map = useRef<L.Map | null>(null)
  useEffect(() => {
    if (!container.current) return
    const m = L.map(container.current, { center: brazil, zoom: 4, scrollWheelZoom: false })
    L.tileLayer(tiles, { attribution, maxZoom: 19 }).addTo(m)
    map.current = m
    return () => {
      m.remove()
      map.current = null
    }
  }, [container])
  return map
}

function pinIcon(label = '') {
  return L.divIcon({
    className: 'map-pin',
    html: `<span>${label}</span>`,
    iconSize: [32, 32],
    iconAnchor: [16, 32],
  })
}

type PinMapProps = {
  value: LatLng | null
  onChange: (p: LatLng) => void
  label: string
}

/** Um pino que a pessoa arrasta ou põe com um toque no mapa. */
export function PinMap({ value, onChange, label }: PinMapProps) {
  const container = useRef<HTMLDivElement>(null)
  const map = useLeaflet(container)
  const marker = useRef<L.Marker | null>(null)
  const change = useRef(onChange)
  useEffect(() => {
    change.current = onChange
  })

  useEffect(() => {
    const m = map.current
    if (!m) return
    const place = (e: L.LeafletMouseEvent) => change.current({ latitude: e.latlng.lat, longitude: e.latlng.lng })
    m.on('click', place)
    return () => void m.off('click', place)
  }, [map])

  useEffect(() => {
    const m = map.current
    if (!m) return
    if (!value) {
      marker.current?.remove()
      marker.current = null
      return
    }
    const at: L.LatLngTuple = [value.latitude, value.longitude]
    if (!marker.current) {
      marker.current = L.marker(at, { draggable: true, icon: pinIcon(), keyboard: false }).addTo(m)
      marker.current.on('dragend', () => {
        const p = marker.current!.getLatLng()
        change.current({ latitude: p.lat, longitude: p.lng })
      })
      m.setView(at, 17)
    } else {
      marker.current.setLatLng(at)
      if (!m.getBounds().contains(at)) m.panTo(at)
    }
  }, [map, value])

  return <div ref={container} role="application" aria-label={label} className="h-64 w-full rounded-lg border border-slate-300" />
}

export type MapStop = { number: number; latitude: number; longitude: number; done: boolean }

/** As paradas da rota numeradas, ligadas na ordem de entrega. */
export function StopsMap({ stops, label }: { stops: MapStop[]; label: string }) {
  const container = useRef<HTMLDivElement>(null)
  const map = useLeaflet(container)

  useEffect(() => {
    const m = map.current
    if (!m || stops.length === 0) return
    const layer = L.layerGroup().addTo(m)
    const points = stops.map((s) => [s.latitude, s.longitude] as L.LatLngTuple)
    L.polyline(points, { color: '#4f46e5', weight: 3, opacity: 0.6 }).addTo(layer)
    for (const s of stops) {
      L.marker([s.latitude, s.longitude], {
        icon: L.divIcon({
          className: s.done ? 'map-stop map-stop-done' : 'map-stop',
          html: `<span>${s.number}</span>`,
          iconSize: [28, 28],
          iconAnchor: [14, 14],
        }),
        keyboard: false,
      }).addTo(layer)
    }
    m.fitBounds(L.latLngBounds(points), { padding: [24, 24], maxZoom: 16 })
    return () => void layer.remove()
  }, [map, stops])

  return <div ref={container} role="img" aria-label={label} className="h-72 w-full rounded-xl border border-slate-200" />
}
