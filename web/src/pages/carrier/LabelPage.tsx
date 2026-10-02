import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router'
import { Button } from '../../components/Button'
import { LoadError, Loading } from '../../components/States'
import { formatCEP, formatPhone } from '../../lib/address'
import { ApiError } from '../../lib/api'
import { useAuth } from '../../lib/auth'
import { getDelivery, type Delivery } from '../../lib/deliveries'
import { useMe } from '../../lib/me'
import { qrDataURL, trackingURL } from '../../lib/qr'

/**
 * Etiqueta de 10 x 15 cm para colar no pacote. O QR-code leva o link de
 * rastreio: o motorista bipa para carregar a rota e o cliente, se ler, vê
 * onde está a entrega.
 */
export function LabelPage() {
  const id = Number(useParams().id)
  const { api } = useAuth()
  const delivery = useQuery({
    queryKey: ['delivery', id],
    queryFn: ({ signal }) => getDelivery(api, id, signal),
    enabled: Number.isInteger(id) && id > 0,
  })

  return (
    <div className="mx-auto max-w-md px-4 py-6 print:m-0 print:max-w-none print:p-0">
      <div className="mb-4 flex items-center justify-between print:hidden">
        <Link to={`/transportadora/entregas/${id}`} className="font-medium text-brand-700 hover:underline">
          ← Entrega
        </Link>
        {delivery.data && <Button onClick={() => window.print()}>Imprimir</Button>}
      </div>
      {!Number.isInteger(id) || id <= 0 || (delivery.error instanceof ApiError && delivery.error.status === 404) ? (
        <p className="text-lg font-semibold">Entrega não encontrada.</p>
      ) : delivery.isPending ? (
        <Loading label="Carregando etiqueta" />
      ) : delivery.isError ? (
        <LoadError error={delivery.error} onRetry={() => void delivery.refetch()} />
      ) : (
        <Label delivery={delivery.data} />
      )}
    </div>
  )
}

function Label({ delivery: d }: { delivery: Delivery }) {
  const me = useMe()
  const qr = useQuery({
    queryKey: ['qr', d.tracking_code],
    queryFn: () => qrDataURL(trackingURL(d.tracking_code)),
    staleTime: Infinity,
  })
  const structured = d.postal_code !== ''
  return (
    <article
      aria-label="Etiqueta"
      className="flex flex-col gap-3 border-2 border-slate-900 bg-white p-4 text-slate-900 print:border-0"
    >
      <header className="flex items-center justify-between border-b-2 border-slate-900 pb-2">
        <span className="text-lg font-bold">{me.data?.carrier.name ?? 'Rastreia'}</span>
        <span className="text-sm">Pedido {d.id}</span>
      </header>
      <div className="flex items-center gap-4">
        {qr.data ? (
          <img src={qr.data} alt={`QR-code da entrega ${d.tracking_code}`} className="size-36 shrink-0" />
        ) : (
          <div className="size-36 shrink-0 bg-slate-100" />
        )}
        <div className="min-w-0">
          <p className="text-sm">Código de rastreio</p>
          <p className="font-mono text-xl font-bold tracking-wider break-all">{d.tracking_code}</p>
        </div>
      </div>
      <section className="border-t-2 border-slate-900 pt-2">
        <p className="text-sm font-semibold uppercase">Destinatário</p>
        <p className="text-xl font-bold">{d.recipient_name}</p>
        {d.recipient_phone && <p>{formatPhone(d.recipient_phone)}</p>}
        {structured ? (
          <>
            <p className="mt-1 text-lg">
              {d.street}, {d.number}
              {d.complement && ` - ${d.complement}`}
            </p>
            <p>
              {d.district} · {d.city} - {d.state}
            </p>
            <p className="text-2xl font-bold tracking-wide">{formatCEP(d.postal_code)}</p>
          </>
        ) : (
          <p className="mt-1 text-lg">{d.address}</p>
        )}
        {d.address_reference && <p className="mt-1 text-sm">Referência: {d.address_reference}</p>}
      </section>
    </article>
  )
}
