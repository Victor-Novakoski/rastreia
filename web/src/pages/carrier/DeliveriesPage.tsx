import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { StatusBadge } from '../../components/StatusBadge'
import { Empty, LoadError, Loading } from '../../components/States'
import { useAuth } from '../../lib/auth'
import { listDeliveries, pageSize, type DriverFilter } from '../../lib/deliveries'
import { formatDateTime } from '../../lib/format'
import { useDrivers } from '../../lib/queries'
import { statuses, statusInfo, type Status } from '../../lib/status'

export function DeliveriesPage() {
  const { api } = useAuth()
  const [params, setParams] = useSearchParams()
  const status = statuses.find((s) => s === params.get('status'))
  const search = params.get('q') ?? ''
  const driver = driverFilter(params.get('driver'))
  const page = Math.max(1, Number(params.get('page')) || 1)

  // O campo acompanha a URL quando ela muda por fora (voltar, link).
  const [text, setText] = useState(search)
  const [shownSearch, setShownSearch] = useState(search)
  if (search !== shownSearch) {
    setShownSearch(search)
    setText(search)
  }

  const deliveries = useQuery({
    queryKey: ['deliveries', { status, search, driver, page }],
    queryFn: ({ signal }) => listDeliveries(api, { status, search, driver, page }, signal),
    placeholderData: keepPreviousData,
  })
  // Enquanto a página nova não chega, a tabela mostra a anterior apagada e
  // os botões esperam: dois cliques em Próxima não pulam uma página.
  const changing = deliveries.isPlaceholderData
  const drivers = useDrivers()
  const driverName = (id: number | null) =>
    id === null ? '—' : (drivers.data?.find((d) => d.id === id)?.name ?? `#${id}`)

  function update(next: { status?: Status | ''; search?: string; driver?: string; page?: number }) {
    setParams((prev) => {
      const p = new URLSearchParams(prev)
      for (const [key, value] of [
        ['status', next.status],
        ['q', next.search],
        ['driver', next.driver],
      ] as const) {
        if (value === undefined) continue
        if (value) p.set(key, value)
        else p.delete(key)
        p.delete('page')
      }
      if (next.page !== undefined) p.set('page', String(next.page))
      return p
    })
  }

  function submitSearch(e: FormEvent) {
    e.preventDefault()
    update({ search: text.trim() })
  }

  const rows = deliveries.data ?? []
  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="text-2xl font-bold">Entregas</h1>
        <Link
          to="/transportadora/entregas/nova"
          className="inline-flex min-h-11 items-center rounded-lg bg-brand-700 px-4 font-semibold text-white hover:bg-brand-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600"
        >
          Nova entrega
        </Link>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-x-6 gap-y-3">
        <form role="search" onSubmit={submitSearch} className="flex grow items-center gap-2 sm:grow-0">
          <label htmlFor="search" className="font-medium">
            Buscar
          </label>
          <input
            id="search"
            type="search"
            value={text}
            maxLength={100}
            placeholder="Código, nome ou e-mail"
            onChange={(e) => {
              setText(e.target.value)
              // Apagar o texto já mostra todas de novo, sem precisar buscar.
              if (!e.target.value && search) update({ search: '' })
            }}
            className="min-h-11 w-full min-w-0 rounded-lg border border-slate-300 bg-white px-3 sm:w-72"
          />
          <button
            type="submit"
            className="min-h-11 shrink-0 rounded-lg border border-slate-300 bg-white px-4 font-semibold text-brand-700 hover:bg-slate-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600"
          >
            Buscar
          </button>
        </form>
        <div className="flex items-center gap-2">
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
        <div className="flex items-center gap-2">
          <label htmlFor="driver-filter" className="font-medium">
            Motorista
          </label>
          <select
            id="driver-filter"
            value={driver ?? ''}
            onChange={(e) => update({ driver: e.target.value })}
            className="min-h-11 rounded-lg border border-slate-300 bg-white px-3"
          >
            <option value="">Todos</option>
            <option value="none">Sem motorista</option>
            {drivers.data?.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
            {typeof driver === 'number' && drivers.data && !drivers.data.some((d) => d.id === driver) && (
              <option value={driver}>#{driver}</option>
            )}
          </select>
        </div>
      </div>

      <div className="mt-4">
        {deliveries.isPending ? (
          <Loading label="Carregando entregas" />
        ) : deliveries.isError ? (
          <LoadError error={deliveries.error} onRetry={() => void deliveries.refetch()} />
        ) : rows.length === 0 ? (
          <Empty>
            {page > 1
              ? 'Não há mais entregas. Volte para a página anterior.'
              : search
                ? `Nenhuma entrega encontrada para “${search}”.`
                : status || driver !== undefined
                  ? 'Nenhuma entrega com esse filtro.'
                  : 'Nenhuma entrega cadastrada ainda.'}
          </Empty>
        ) : (
          <div
            aria-busy={changing}
            className={`overflow-x-auto rounded-lg border border-slate-200 bg-white transition-opacity ${changing ? 'opacity-60' : ''}`}
          >
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
                        to={`/transportadora/entregas/${d.id}`}
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
          disabled={page <= 1 || changing}
          onClick={() => update({ page: page - 1 })}
          className="min-h-11 rounded-lg border border-slate-300 bg-white px-4 font-medium disabled:opacity-50"
        >
          Anterior
        </button>
        <span className="text-sm text-slate-600">Página {page}</span>
        <button
          type="button"
          // A API não devolve o total: página incompleta é a última.
          disabled={rows.length < pageSize || changing}
          onClick={() => update({ page: page + 1 })}
          className="min-h-11 rounded-lg border border-slate-300 bg-white px-4 font-medium disabled:opacity-50"
        >
          Próxima
        </button>
      </nav>
    </>
  )
}

/** O filtro de motorista da URL: "none" ou um id; qualquer outra coisa é ignorada. */
function driverFilter(value: string | null): DriverFilter | undefined {
  if (value === 'none') return 'none'
  return value && /^[1-9]\d*$/.test(value) ? Number(value) : undefined
}
