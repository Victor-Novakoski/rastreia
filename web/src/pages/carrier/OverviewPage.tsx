import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router'
import { Alert } from '../../components/Alert'
import { StatusIcon } from '../../components/StatusIcon'
import { LoadError, Loading } from '../../components/States'
import { useAuth } from '../../lib/auth'
import { getSummary } from '../../lib/deliveries'
import { useMe } from '../../lib/me'
import { useDrivers } from '../../lib/queries'
import { statuses, statusInfo, toneClasses } from '../../lib/status'

const action =
  'inline-flex min-h-11 items-center justify-center rounded-lg px-4 font-semibold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600'

/** Visão geral da transportadora: números do mês, o que pede atenção e primeiros passos. */
export function OverviewPage() {
  const { api } = useAuth()
  const location = useLocation()
  const me = useMe()
  const drivers = useDrivers()
  const summary = useQuery({ queryKey: ['deliveries', 'summary'], queryFn: ({ signal }) => getSummary(api, signal) })
  const welcome = (location.state as { welcome?: boolean } | null)?.welcome

  if (summary.isPending || drivers.isPending) return <Loading label="Carregando visão geral" />
  if (summary.isError) return <LoadError error={summary.error} onRetry={() => void summary.refetch()} />
  if (drivers.isError) return <LoadError error={drivers.error} onRetry={() => void drivers.refetch()} />

  const counts = summary.data.by_status
  const total = statuses.reduce((n, s) => n + counts[s], 0)
  const open = counts.pending + counts.picked_up + counts.in_transit
  const firstName = me.data?.name.split(' ')[0]

  return (
    <>
      {welcome && <Alert tone="success">Conta criada. Siga os primeiros passos abaixo para lançar sua primeira entrega.</Alert>}
      <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{firstName ? `Olá, ${firstName}` : 'Visão geral'}</h1>
          <p className="text-slate-600">Entregas criadas nos últimos 30 dias.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link to="/transportadora/motoristas" className={`${action} border border-slate-300 bg-white hover:bg-slate-50`}>
            Cadastrar motorista
          </Link>
          <Link to="/transportadora/entregas/nova" className={`${action} bg-brand-700 text-white hover:bg-brand-800`}>
            Nova entrega
          </Link>
        </div>
      </div>

      {(drivers.data.length === 0 || total === 0) && <FirstSteps hasDrivers={drivers.data.length > 0} />}

      <section aria-labelledby="numbers" className="mt-6">
        <h2 id="numbers" className="sr-only">
          Números
        </h2>
        <div className="grid gap-3 sm:grid-cols-3">
          <Stat label="Em andamento" value={open} to="/transportadora/entregas" />
          <Stat label="Entregues" value={counts.delivered} to="/transportadora/entregas?status=delivered" tone="success" />
          <Stat
            label="Sem motorista"
            value={summary.data.unassigned}
            to="/transportadora/entregas?status=pending"
            tone={summary.data.unassigned > 0 ? 'warning' : undefined}
            hint={summary.data.unassigned > 0 ? 'Aguardando alguém para coletar' : 'Tudo distribuído'}
          />
        </div>
        <ul className="mt-3 grid gap-3 sm:grid-cols-5">
          {statuses.map((s) => (
            <li key={s}>
              <Link
                to={`/transportadora/entregas?status=${s}`}
                className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white p-3 hover:border-brand-600 focus-visible:outline-2 focus-visible:outline-brand-600"
              >
                <span className={`flex size-9 items-center justify-center rounded-full ${toneClasses[statusInfo[s].tone]}`}>
                  <StatusIcon status={s} className="size-4" />
                </span>
                <span>
                  <span className="block text-xl font-bold">{counts[s]}</span>
                  <span className="block text-sm text-slate-600">{statusInfo[s].label}</span>
                </span>
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </>
  )
}

function Stat({
  label,
  value,
  to,
  tone,
  hint,
}: {
  label: string
  value: number
  to: string
  tone?: 'success' | 'warning'
  hint?: string
}) {
  const color = tone === 'success' ? 'text-success-fg' : tone === 'warning' ? 'text-danger-fg' : 'text-slate-900'
  return (
    <Link
      to={to}
      className="rounded-xl border border-slate-200 bg-white p-5 hover:border-brand-600 focus-visible:outline-2 focus-visible:outline-brand-600"
    >
      <span className="block text-sm font-medium text-slate-600">{label}</span>
      <span className={`mt-1 block text-3xl font-bold ${color}`}>{value}</span>
      {hint && <span className="mt-1 block text-sm text-slate-600">{hint}</span>}
    </Link>
  )
}

function FirstSteps({ hasDrivers }: { hasDrivers: boolean }) {
  return (
    <section aria-labelledby="first-steps" className="mt-6 rounded-xl border border-brand-600/30 bg-brand-50 p-5">
      <h2 id="first-steps" className="text-lg font-bold text-brand-800">
        Primeiros passos
      </h2>
      <ol className="mt-3 flex flex-col gap-3">
        <Step done={hasDrivers} n={1}>
          <Link to="/transportadora/motoristas" className="font-semibold text-brand-700 underline-offset-2 hover:underline">
            Cadastre um motorista
          </Link>{' '}
          e passe para ele o e-mail, a senha e o endereço do app do motorista.
        </Step>
        <Step done={false} n={2}>
          <Link to="/transportadora/entregas/nova" className="font-semibold text-brand-700 underline-offset-2 hover:underline">
            Lance a primeira entrega
          </Link>{' '}
          e escolha quem vai levar.
        </Step>
        <Step done={false} n={3}>
          O cliente recebe o código por e-mail e acompanha cada passo pela página de rastreio.
        </Step>
      </ol>
    </section>
  )
}

function Step({ done, n, children }: { done: boolean; n: number; children: ReactNode }) {
  return (
    <li className="flex gap-3">
      <span
        className={`flex size-7 shrink-0 items-center justify-center rounded-full text-sm font-bold ${done ? 'bg-success-fg text-white' : 'bg-white text-brand-800'}`}
      >
        {done ? '✓' : n}
        <span className="sr-only">{done ? ' (feito)' : ''}</span>
      </span>
      <p className={done ? 'text-slate-500 line-through' : ''}>{children}</p>
    </li>
  )
}
