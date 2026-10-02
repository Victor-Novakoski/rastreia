import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router'
import { StatusBadge } from '../../components/StatusBadge'
import { Empty, LoadError, Loading } from '../../components/States'
import { useAuth } from '../../lib/auth'
import { listDeliveries, pageSize } from '../../lib/deliveries'
import { formatDateTime } from '../../lib/format'
import { useDrivers } from '../../lib/queries'
import { statuses, statusInfo, type Status } from '../../lib/status'

export function DeliveriesPage() {
  const { api } = useAuth()
  const [params, setParams] = useSearchParams()
  const status = statuses.find((s) => s === params.get('status'))
  const page = Math.max(1, Number(params.get('page')) || 1)

  const deliveries = useQuery({
    queryKey: ['deliveries', { status, page }],
    queryFn: ({ signal }) => listDeliveries(api, { status, page }, signal),
    placeholderData: keepPreviousData,
  })
  const drivers = useDrivers()
  const driverName = (id: number | null) =>
    id === null ? '—' : (drivers.data?.find((d) => d.id === id)?.name ?? `#${id}`)

  function update(next: { status?: Status | ''; page?: number }) {
    const p = new URLSearchParams(params)
    if (next.status !== undefined) {
      if (next.status) p.set('status', next.status)
      else p.delete('status')
      p.delete('page')
    }
    if (next.page !== undefined) p.set('page', String(next.page))
    setParams(p)
  }

  const rows = deliveries.data ?? []
  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="text-2xl font-bold">Entregas</h1>
        <Link
          to="/admin/entregas/nova"
          className="inline-flex min-h-11 items-center rounded-lg bg-brand-700 px-4 font-semibold text-white hover:bg-brand-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600"
        >
          Nova entrega
        </Link>
      </div>

      <div className="mt-4 flex items-center gap-2">
        <label htmlFor="status-filter" className="font-medium">
          Status
        </label>
        <select
          id="status-filter"
          value={status ?? ''}
          onChange={(e) => update({ status: e.target.value as Status | '' })}
          className="min-h-11 rounded-lg border border-slate-300 bg-white px-3"
        >
          <option value="">Todos</option>
          {statuses.map((s) => (
            <option key={s} value={s}>
              {statusInfo[s].label}
            </option>
          ))}
        </select>
      </div>

      <div className="mt-4">
        {deliveries.isPending ? (
          <Loading label="Carregando entregas" />
        ) : deliveries.isError ? (
          <LoadError error={deliveries.error} onRetry={() => void deliveries.refetch()} />
        ) : rows.length === 0 ? (
          <Empty>{status || page > 1 ? 'Nenhuma entrega com esse filtro.' : 'Nenhuma entrega cadastrada ainda.'}</Empty>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                <tr>
                  <th className="px-3 py-2 font-semibold">Código</th>
                  <th className="px-3 py-2 font-semibold">Destinatário</th>
                  <th className="px-3 py-2 font-semibold">Motorista</th>
                  <th className="px-3 py-2 font-semibold">Status</th>
                  <th className="px-3 py-2 font-semibold">Atualizada</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((d) => (
                  <tr key={d.id} className="border-b border-slate-100 last:border-0 hover:bg-slate-50">
                    <td className="px-3 py-2">
                      <Link
                        to={`/admin/entregas/${d.id}`}
                        className="font-mono font-semibold text-brand-700 underline-offset-2 hover:underline"
                      >
                        {d.tracking_code}
                      </Link>
                    </td>
                    <td className="px-3 py-2">{d.recipient_name}</td>
                    <td className="px-3 py-2">{driverName(d.driver_id)}</td>
                    <td className="px-3 py-2">
                      <StatusBadge status={d.status} />
                    </td>
                    <td className="px-3 py-2 whitespace-nowrap">{formatDateTime(d.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <nav className="mt-4 flex items-center justify-between" aria-label="Paginação">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => update({ page: page - 1 })}
          className="min-h-11 rounded-lg border border-slate-300 bg-white px-4 font-medium disabled:opacity-50"
        >
          Anterior
        </button>
        <span className="text-sm text-slate-600">Página {page}</span>
        <button
          type="button"
          // A API não devolve o total: página incompleta é a última.
          disabled={rows.length < pageSize}
          onClick={() => update({ page: page + 1 })}
          className="min-h-11 rounded-lg border border-slate-300 bg-white px-4 font-medium disabled:opacity-50"
        >
          Próxima
        </button>
      </nav>
    </>
  )
}
