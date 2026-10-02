import { Link } from 'react-router'
import { StatusBadge } from '../../components/StatusBadge'
import { Empty, LoadError, Loading } from '../../components/States'
import type { Delivery } from '../../lib/deliveries'
import { useMyDeliveries } from '../../lib/driver'
import { formatDateTime } from '../../lib/format'

export function DriverDeliveriesPage() {
  const deliveries = useMyDeliveries()
  if (deliveries.isPending) return <Loading label="Carregando entregas" />
  if (deliveries.isError) return <LoadError error={deliveries.error} onRetry={() => void deliveries.refetch()} />

  // Pendentes primeiro; as concluídas ficam embaixo, para consulta.
  const open = deliveries.data.filter((d) => d.status !== 'delivered')
  const done = deliveries.data.filter((d) => d.status === 'delivered')
  return (
    <>
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Para fazer ({open.length})</h1>
        <button
          type="button"
          onClick={() => void deliveries.refetch()}
          disabled={deliveries.isFetching}
          className="min-h-11 rounded-lg px-3 font-semibold text-brand-700 disabled:opacity-60"
        >
          {deliveries.isFetching ? 'Atualizando…' : 'Atualizar'}
        </button>
      </div>
      {open.length === 0 ? (
        <div className="mt-3">
          <Empty>Nenhuma entrega pendente.</Empty>
        </div>
      ) : (
        <ul className="mt-3 flex flex-col gap-3">
          {open.map((d) => (
            <DeliveryCard key={d.id} delivery={d} />
          ))}
        </ul>
      )}
      {done.length > 0 && (
        <details className="mt-6">
          <summary className="min-h-11 cursor-pointer py-2 font-semibold">Entregues ({done.length})</summary>
          <ul className="mt-2 flex flex-col gap-3">
            {done.map((d) => (
              <DeliveryCard key={d.id} delivery={d} />
            ))}
          </ul>
        </details>
      )}
    </>
  )
}

function DeliveryCard({ delivery: d }: { delivery: Delivery }) {
  return (
    <li>
      <Link
        to={`/motorista/entregas/${d.id}`}
        className="block rounded-xl border border-slate-200 bg-white p-4 shadow-sm active:bg-slate-50 focus-visible:outline-2 focus-visible:outline-brand-600"
      >
        <div className="flex items-start justify-between gap-2">
          <p className="text-lg font-semibold">{d.recipient_name}</p>
          <StatusBadge status={d.status} />
        </div>
        <p className="mt-1 text-slate-700">{d.address}</p>
        <p className="mt-2 text-sm text-slate-600">
          <span className="font-mono">{d.tracking_code}</span> · {formatDateTime(d.updated_at)}
        </p>
      </Link>
    </li>
  )
}
