import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState, type FormEvent, type PointerEvent } from 'react'
import { Link } from 'react-router'
import { Alert } from '../../components/Alert'
import { Button } from '../../components/Button'
import { StopsMap } from '../../components/LazyMap'
import type { MapStop } from '../../components/Map'
import { QrScanner } from '../../components/QrScanner'
import { StatusBadge } from '../../components/StatusBadge'
import { Empty, LoadError, Loading } from '../../components/States'
import { formatPhone } from '../../lib/address'
import { ApiError, errorMessage } from '../../lib/api'
import { useAuth } from '../../lib/auth'
import {
  addToRoute,
  currentPosition,
  optimizeRoute,
  packageIDs,
  removeFromRoute,
  reorderRoute,
  routeKey,
  useRoute,
  type Route,
  type Stop,
} from '../../lib/route'

/** Rota do dia: bipar os pacotes, ver as paradas no mapa e escolher a ordem. */
export function RoutePage() {
  const route = useRoute()
  if (route.isPending) return <Loading label="Carregando rota" />
  if (route.isError) return <LoadError error={route.error} onRetry={() => void route.refetch()} />
  return <RouteView route={route.data} />
}

type Notice = { tone: 'success' | 'danger'; text: string } | null

function RouteView({ route }: { route: Route }) {
  const { api } = useAuth()
  const queryClient = useQueryClient()
  const [scanning, setScanning] = useState(false)
  const [code, setCode] = useState('')
  const [notice, setNotice] = useState<Notice>(null)

  const saved = (r: Route) => queryClient.setQueryData(routeKey, r)

  const add = useMutation({
    mutationFn: (text: string) => addToRoute(api, text),
    onSuccess: (r, text) => {
      saved(r)
      // Pacote sem motorista passa a ser deste motorista.
      void queryClient.invalidateQueries({ queryKey: ['me', 'deliveries'] })
      const p = r.stops.flatMap((s) => s.packages.map((pk) => ({ stop: s.number, pk }))).find((x) => text.toUpperCase().endsWith(x.pk.tracking_code))
      setNotice({
        tone: 'success',
        text: p ? `Pacote ${p.pk.position} carregado (parada ${p.stop}): ${p.pk.recipient_name}.` : 'Pacote carregado.',
      })
      setCode('')
    },
    onError: (err) => setNotice({ tone: 'danger', text: scanError(err) }),
  })

  const optimize = useMutation({
    mutationFn: async () => optimizeRoute(api, await currentPosition()),
    onSuccess: (r) => {
      saved(r)
      setNotice({ tone: 'success', text: 'Rota organizada. Mude a ordem se preferir.' })
    },
    onError: (err) => setNotice({ tone: 'danger', text: errorMessage(err) }),
  })

  /**
   * A rota mudou em outro lugar (outro aparelho, ou a aba ficou aberta de um
   * dia para o outro): a ordem enviada já não bate com a do servidor.
   */
  function changeFailed(err: unknown) {
    if (err instanceof ApiError && (err.status === 404 || err.status === 422)) {
      void queryClient.invalidateQueries({ queryKey: routeKey })
      setNotice({ tone: 'danger', text: 'A rota mudou. Atualizamos a lista, confira e tente de novo.' })
      return
    }
    setNotice({ tone: 'danger', text: errorMessage(err) })
  }

  const reorder = useMutation({
    mutationFn: (stops: Stop[]) => reorderRoute(api, packageIDs(stops)),
    onMutate: (stops) => {
      const before = queryClient.getQueryData<Route>(routeKey)
      saved({ ...route, stops: renumber(stops) })
      return before
    },
    onSuccess: saved,
    onError: (err, _, before) => {
      if (before) saved(before)
      changeFailed(err)
    },
  })

  const remove = useMutation({
    mutationFn: (id: number) => removeFromRoute(api, id),
    onSuccess: saved,
    onError: changeFailed,
  })

  function submit(e: FormEvent) {
    e.preventDefault()
    if (code.trim() && !add.isPending) add.mutate(code.trim())
  }

  const mapStops = useMemo<MapStop[]>(
    () =>
      route.stops
        .filter((s) => s.latitude != null && s.longitude != null)
        .map((s) => ({ number: s.number, latitude: s.latitude!, longitude: s.longitude!, done: finished(s) })),
    [route.stops],
  )
  const offMap = route.stops.length - mapStops.length

  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-xl font-bold">Rota de hoje</h1>
        <p className="text-slate-600">
          {route.stops.length} {route.stops.length === 1 ? 'parada' : 'paradas'} · {route.total_packages}{' '}
          {route.total_packages === 1 ? 'pacote' : 'pacotes'}
        </p>
      </div>

      <section aria-label="Carregar pacotes" className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4">
        {scanning ? (
          <QrScanner onCode={(text) => add.mutate(text)} onClose={() => setScanning(false)} />
        ) : (
          <Button onClick={() => setScanning(true)} className="w-full text-lg">
            Bipar pacotes
          </Button>
        )}
        <form onSubmit={submit} className="flex gap-2">
          <label htmlFor="route-code" className="sr-only">
            Código do pacote
          </label>
          <input
            id="route-code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="Ou digite o código"
            autoCapitalize="characters"
            autoComplete="off"
            className="min-h-11 w-full min-w-0 rounded-lg border border-slate-300 px-3 font-mono uppercase placeholder:font-sans placeholder:normal-case focus:border-brand-600 focus:outline-2 focus:outline-brand-600"
          />
          <Button type="submit" variant="secondary" loading={add.isPending}>
            Adicionar
          </Button>
        </form>
        {notice && <Alert tone={notice.tone}>{notice.text}</Alert>}
      </section>

      {route.stops.length === 0 ? (
        <Empty>Nenhum pacote na rota. Bipe as etiquetas para montar a rota do dia.</Empty>
      ) : (
        <>
          <div className="flex flex-col gap-2">
            <Button variant="secondary" onClick={() => optimize.mutate()} loading={optimize.isPending}>
              Organizar melhor rota
            </Button>
            <p className="text-sm text-slate-600">
              Começa pela parada mais perto de você. Depois é só arrastar para mudar a ordem.
            </p>
          </div>
          {mapStops.length > 0 && <StopsMap stops={mapStops} label="Mapa com as paradas numeradas" />}
          {offMap > 0 && (
            <p className="text-sm text-slate-600">
              {offMap === 1 ? '1 parada não tem' : `${offMap} paradas não têm`} ponto no mapa e {offMap === 1 ? 'fica' : 'ficam'} no fim
              da rota organizada.
            </p>
          )}
          <StopList
            stops={route.stops}
            onReorder={(stops) => reorder.mutate(stops)}
            onRemove={(id) => remove.mutate(id)}
            busy={remove.isPending}
          />
        </>
      )}
    </div>
  )
}

function scanError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 404) return 'Pacote não encontrado. Confira o código.'
    if (err.status === 409) {
      if (err.message.includes('another driver')) return 'Este pacote é de outro motorista.'
      if (err.message.includes('finished')) return 'Este pacote já foi entregue.'
      return 'A rota está cheia.'
    }
    if (err.status === 0) return 'Sem sinal. Bipe de novo quando voltar.'
  }
  return errorMessage(err)
}

function finished(s: Stop) {
  return s.packages.every((p) => p.status === 'delivered')
}

/** Números das paradas e dos pacotes de novo, na ordem da lista. */
function renumber(stops: Stop[]): Stop[] {
  let n = 0
  return stops.map((s, i) => ({ ...s, number: i + 1, packages: s.packages.map((p) => ({ ...p, position: ++n })) }))
}

function move<T>(list: T[], from: number, to: number): T[] {
  const out = [...list]
  const [item] = out.splice(from, 1)
  out.splice(to, 0, item)
  return out
}

type StopListProps = {
  stops: Stop[]
  onReorder: (stops: Stop[]) => void
  onRemove: (deliveryID: number) => void
  busy: boolean
}

/** offset vai do dedo ao meio do cartão arrastado. */
type Drag = { index: number; list: Stop[]; offset: number }

/**
 * Lista das paradas. Arrastar pela alça funciona no toque e no mouse (eventos
 * de ponteiro, porque o arrastar do HTML não funciona no celular); as setas
 * fazem o mesmo para quem não consegue arrastar.
 */
function StopList({ stops, onReorder, onRemove, busy }: StopListProps) {
  const [drag, setDrag] = useState<Drag | null>(null)
  // O último estado do arraste, para os eventos da janela, que não veem o
  // estado novo do React.
  const current = useRef<Drag | null>(null)
  const stopListening = useRef<(() => void) | null>(null)
  const items = useRef<(HTMLLIElement | null)[]>([])
  const shown = drag?.list ?? stops

  useEffect(() => () => stopListening.current?.(), [])

  function update(next: Drag | null) {
    current.current = next
    setDrag(next)
  }

  function start(e: PointerEvent, index: number) {
    if (current.current) return // um dedo de cada vez
    const card = items.current[index]?.getBoundingClientRect()
    if (!card) return
    e.currentTarget.setPointerCapture(e.pointerId)
    update({ index, list: stops, offset: card.top + card.height / 2 - e.clientY })
    // Ao descer, o React move no DOM o próprio cartão arrastado e o navegador
    // perde a captura do ponteiro; ouvir na janela segue o dedo até o fim.
    const pointer = e.pointerId
    const onMove = (ev: globalThis.PointerEvent) => {
      if (ev.pointerId === pointer) over(ev.clientY)
    }
    const onUp = (ev: globalThis.PointerEvent) => {
      if (ev.pointerId === pointer) end()
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onUp)
    stopListening.current = () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onUp)
      stopListening.current = null
    }
  }

  /**
   * O cartão vai para o lugar cujo meio fica mais perto do meio dele sob o
   * dedo. Os lugares saem da altura dos outros cartões, que não muda ao
   * trocar, então a troca não vai e volta quando as alturas diferem.
   */
  function over(y: number) {
    const d = current.current
    if (!d) return
    const rects = d.list.map((_, i) => items.current[i]?.getBoundingClientRect())
    if (rects.some((r) => !r)) return
    const boxes = rects as DOMRect[]
    const gap = boxes.length > 1 ? boxes[1].top - boxes[0].bottom : 0
    const own = boxes[d.index].height
    const others = boxes.filter((_, i) => i !== d.index)
    const center = y + d.offset
    let target = 0
    let top = boxes[0].top // onde a lista começa
    for (const other of others) {
      // Entre o meio do cartão neste lugar e no próximo.
      if (center < top + own / 2 + (other.height + gap) / 2) break
      target++
      top += other.height + gap
    }
    if (target !== d.index) update({ ...d, index: target, list: move(d.list, d.index, target) })
  }

  function end() {
    stopListening.current?.()
    const d = current.current
    if (!d) return
    update(null)
    if (d.list.some((s, i) => s !== stops[i])) onReorder(d.list)
  }

  return (
    <ol className="flex flex-col gap-3" aria-label="Paradas">
      {shown.map((s, i) => (
        <li
          key={s.packages[0]?.id ?? s.number}
          ref={(el) => {
            items.current[i] = el
          }}
          className={`rounded-xl border bg-white p-3 shadow-sm ${drag?.index === i ? 'border-brand-600 ring-2 ring-brand-600' : 'border-slate-200'} ${finished(s) ? 'opacity-70' : ''}`}
        >
          <div className="flex items-start gap-3">
            <span
              className={`flex size-9 shrink-0 items-center justify-center rounded-full text-lg font-bold text-white ${finished(s) ? 'bg-success-fg' : 'bg-brand-700'}`}
              aria-label={`Parada ${drag ? i + 1 : s.number}`}
            >
              {drag ? i + 1 : s.number}
            </span>
            <div className="min-w-0 flex-1">
              <p className="font-semibold">{s.address}</p>
              <MapLink stop={s} />
            </div>
            <div className="flex shrink-0 items-center">
              <div className="flex flex-col">
                <button
                  type="button"
                  aria-label={`Subir parada ${s.number}`}
                  disabled={i === 0 || !!drag}
                  onClick={() => onReorder(move(stops, i, i - 1))}
                  className="flex size-11 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 disabled:opacity-30"
                >
                  ▲
                </button>
                <button
                  type="button"
                  aria-label={`Descer parada ${s.number}`}
                  disabled={i === shown.length - 1 || !!drag}
                  onClick={() => onReorder(move(stops, i, i + 1))}
                  className="flex size-11 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 disabled:opacity-30"
                >
                  ▼
                </button>
              </div>
              <span
                role="button"
                tabIndex={-1}
                aria-label={`Arrastar parada ${s.number}`}
                onPointerDown={(e) => start(e, i)}
                className="flex h-22 w-10 cursor-grab touch-none items-center justify-center text-xl text-slate-400 select-none active:cursor-grabbing"
              >
                ⠿
              </span>
            </div>
          </div>
          <ul className="mt-2 flex flex-col gap-2 border-t border-slate-100 pt-2">
            {s.packages.map((p) => (
              <li key={p.id} className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  <span className="rounded-md bg-brand-50 px-2 py-0.5 font-mono text-sm font-bold text-brand-800">
                    #{p.position}
                  </span>
                  <Link
                    to={`/motorista/entregas/${p.id}`}
                    className="flex min-h-11 min-w-0 flex-1 items-center font-medium text-brand-700 underline-offset-2 hover:underline"
                  >
                    {p.recipient_name}
                  </Link>
                </div>
                <div>
                  <StatusBadge status={p.status} />
                </div>
                {(p.complement || p.address_reference) && (
                  <p className="text-sm text-slate-600">{[p.complement, p.address_reference].filter(Boolean).join(' · ')}</p>
                )}
                <div className="flex flex-wrap items-center gap-x-4 text-sm">
                  {p.recipient_phone && (
                    <a href={`tel:+55${p.recipient_phone}`} className="inline-flex min-h-11 items-center font-medium text-brand-700">
                      Ligar {formatPhone(p.recipient_phone)}
                    </a>
                  )}
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onRemove(p.id)}
                    className="inline-flex min-h-11 items-center font-medium text-slate-600 disabled:opacity-50"
                  >
                    Tirar da rota
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </li>
      ))}
    </ol>
  )
}

/** Abre a navegação no app de mapas do celular, pelo ponto ou pelo endereço. */
function MapLink({ stop: s }: { stop: Stop }) {
  const query = s.latitude != null && s.longitude != null ? `${s.latitude},${s.longitude}` : s.address
  return (
    <a
      href={`https://www.google.com/maps/dir/?api=1&destination=${encodeURIComponent(query)}`}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex min-h-11 items-center text-sm font-medium text-brand-700 underline underline-offset-2"
    >
      Navegar até aqui
    </a>
  )
}
