import { useEffect, useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import { PublicLayout } from '../components/PublicLayout'
import { Spinner } from '../components/Spinner'
import { StatusBadge } from '../components/StatusBadge'
import { LiveBadge } from '../components/LiveBadge'
import { StatusIcon } from '../components/StatusIcon'
import { ApiError } from '../lib/api'
import { formatDateTime } from '../lib/format'
import { statusInfo, toneClasses } from '../lib/status'
import { getTracking, isValidCode, normalizeCode, type Tracking } from '../lib/tracking'
import { useLive } from '../lib/useLive'

type State =
  | { kind: 'loading' }
  | { kind: 'ok'; tracking: Tracking }
  | { kind: 'not-found' }
  | { kind: 'error'; message: string }

export function TrackingPage() {
  const code = normalizeCode(useParams().code ?? '')
  const valid = isValidCode(code)
  const [attempt, setAttempt] = useState(0)
  // Resultado da última busca, marcado com o código e a tentativa que o geraram.
  const [result, setResult] = useState<{ key: string; state: State } | null>(null)
  const key = `${code}#${attempt}`

  useEffect(() => {
    if (!valid) return
    const ctrl = new AbortController()
    getTracking(code, ctrl.signal)
      .then((tracking) => setResult({ key, state: { kind: 'ok', tracking } }))
      .catch((err: unknown) => {
        if (ctrl.signal.aborted) return
        setResult({ key, state: toErrorState(err) })
      })
    return () => ctrl.abort()
  }, [code, key, valid])

  // Código malformado nem chega a ir para a API: a resposta seria o mesmo 404.
  const state: State = !valid
    ? { kind: 'not-found' }
    : result?.key === key
      ? result.state
      : { kind: 'loading' }

  // Depois de carregar, cada mudança chega pelo WebSocket já no formato do GET.
  const showTracking = (tracking: Tracking) => setResult({ key, state: { kind: 'ok', tracking } })
  const live = useLive(state.kind === 'ok' ? `/public/tracking/${encodeURIComponent(code)}/live` : null, {
    onMessage: (data) => showTracking(data as Tracking),
    // Durante a queda alguma mudança pode ter passado sem aviso.
    onOpen: (again) => {
      if (again) getTracking(code).then(showTracking, () => {})
    },
  })

  return (
    <PublicLayout>
      <p className="font-mono text-sm tracking-wider text-slate-600">{code}</p>
      {state.kind === 'loading' && (
        <div className="mt-8 flex justify-center text-slate-600">
          <Spinner label="Carregando rastreio" />
        </div>
      )}
      {state.kind === 'not-found' && (
        <Message title="Não encontramos essa entrega">
          Confira o código no e-mail que você recebeu. O link deixa de funcionar 30 dias depois da entrega.
        </Message>
      )}
      {state.kind === 'error' && (
        <Message title="Não foi possível carregar">
          {state.message}{' '}
          <button
            type="button"
            onClick={() => setAttempt((n) => n + 1)}
            className="font-semibold text-brand-700 underline underline-offset-2"
          >
            Tentar de novo
          </button>
        </Message>
      )}
      {state.kind === 'ok' && <TrackingDetails tracking={state.tracking} live={live} />}
    </PublicLayout>
  )
}

function toErrorState(err: unknown): State {
  if (err instanceof ApiError && err.status === 404) return { kind: 'not-found' }
  if (err instanceof ApiError && err.status === 429) {
    return { kind: 'error', message: 'Muitas consultas seguidas. Espere um minuto.' }
  }
  if (err instanceof ApiError && err.status === 0) {
    return { kind: 'error', message: 'Sem conexão com o servidor.' }
  }
  return { kind: 'error', message: 'Algo deu errado do nosso lado.' }
}

function TrackingDetails({ tracking, live }: { tracking: Tracking; live: boolean }) {
  const events = [...tracking.events].sort((a, b) => b.created_at.localeCompare(a.created_at))
  return (
    <>
      <h1 className="mt-1 text-2xl font-bold">Olá, {tracking.recipient_first_name}</h1>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <StatusBadge status={tracking.status} size="lg" />
        {live && <LiveBadge />}
      </div>
      <p className="mt-2 text-sm text-slate-600">Atualizado em {formatDateTime(tracking.updated_at)}</p>

      <h2 className="mt-8 text-lg font-semibold">Histórico</h2>
      <ol className="mt-3">
        {events.map((ev, i) => (
          <li key={`${ev.status}-${ev.created_at}`} className="relative flex gap-3 pb-6 last:pb-0">
            {i < events.length - 1 && (
              <span className="absolute top-9 bottom-0 left-4 w-px bg-slate-300" aria-hidden="true" />
            )}
            <span
              className={`flex size-8 shrink-0 items-center justify-center rounded-full ${toneClasses[statusInfo[ev.status].tone]}`}
            >
              <StatusIcon status={ev.status} className="size-4" />
            </span>
            <div className="pt-1">
              <p className="font-medium">{statusInfo[ev.status].label}</p>
              <p className="text-sm text-slate-600">{formatDateTime(ev.created_at)}</p>
            </div>
          </li>
        ))}
      </ol>
    </>
  )
}

function Message({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="mt-6 rounded-lg border border-slate-200 bg-white p-4">
      <h1 className="text-lg font-semibold">{title}</h1>
      <p className="mt-1 text-slate-600">{children}</p>
      <Link to="/rastreio" className="mt-3 inline-block font-semibold text-brand-700 underline underline-offset-2">
        Buscar outro código
      </Link>
    </div>
  )
}
