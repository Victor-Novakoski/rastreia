import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { useEffect, useId, useRef } from 'react'
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

/**
 * Arrastando o mapa para o lado, o Leaflet passa de 180° de longitude (o
 * mundo se repete); wrap traz o ponto de volta para o intervalo que vale.
 */
function point(at: L.LatLng): LatLng {
  const p = at.wrap()
  return { latitude: p.lat, longitude: p.lng }
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

/**
 * Um pino que a pessoa arrasta ou põe com um toque no mapa. Pelo teclado, as
 * setas movem o mapa e Enter põe o pino no centro, marcado por uma mira.
 */
export function PinMap({ value, onChange, label }: PinMapProps) {
  const container = useRef<HTMLDivElement>(null)
  const map = useLeaflet(container)
  const marker = useRef<L.Marker | null>(null)
  const hint = useId()
  const change = useRef(onChange)
  useEffect(() => {
    change.current = onChange
  })

  useEffect(() => {
    const m = map.current
    if (!m) return
    const place = (e: L.LeafletMouseEvent) => change.current(point(e.latlng))
    const placeAtCenter = (e: KeyboardEvent) => {
      // Só no próprio mapa: Enter num botão de zoom continua sendo o botão.
      if (e.key !== 'Enter' || e.target !== m.getContainer()) return
      e.preventDefault()
      change.current(point(m.getCenter()))
    }
    m.on('click', place)
    m.getContainer().addEventListener('keydown', placeAtCenter)
    return () => {
      m.off('click', place)
      m.getContainer().removeEventListener('keydown', placeAtCenter)
    }
  }, [map])

  // O pino sai junto com o mapa. Sem isso, no ensaio de montar duas vezes do
  // StrictMode, o pino ficaria preso ao primeiro mapa e não apareceria.
  useEffect(
    () => () => {
      marker.current?.remove()
      marker.current = null
    },
    [map],
  )

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
      marker.current.on('dragend', () => change.current(point(marker.current!.getLatLng())))
      m.setView(at, 17)
    } else {
      marker.current.setLatLng(at)
      if (!m.getBounds().contains(at)) m.panTo(at)
    }
  }, [map, value])

  return (
    <>
      <div
        ref={container}
        role="application"
        aria-label={label}
        aria-describedby={hint}
        className="pin-map h-64 w-full rounded-lg border border-slate-300"
      />
      <p id={hint} className="sr-only">
        Use as setas para mover o mapa e Enter para pôr o pino no centro.
      </p>
    </>
  )
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
        // Parada feita leva um ✓ no lugar do número, não só outra cor.
        icon: L.divIcon({
          className: s.done ? 'map-stop map-stop-done' : 'map-stop',
          html: `<span>${s.done ? '✓' : s.number}</span>`,
          iconSize: [28, 28],
          iconAnchor: [14, 14],
        }),
        keyboard: false,
      }).addTo(layer)
    }
    m.fitBounds(L.latLngBounds(points), { padding: [24, 24], maxZoom: 16 })
    return () => void layer.remove()
  }, [map, stops])

  // region, não img: lá dentro há os botões de zoom, que recebem foco.
  return <div ref={container} role="region" aria-label={label} className="h-72 w-full rounded-xl border border-slate-200" />
}
