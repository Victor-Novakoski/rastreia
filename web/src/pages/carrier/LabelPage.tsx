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
  const valid = Number.isInteger(id) && id > 0
  const { api } = useAuth()
  const delivery = useQuery({
    queryKey: ['delivery', id],
    queryFn: ({ signal }) => getDelivery(api, id, signal),
    enabled: valid,
  })
  const me = useMe()
  const code = delivery.data?.tracking_code
  const qr = useQuery({
    queryKey: ['qr', code],
    queryFn: () => qrDataURL(trackingURL(code!)),
    enabled: code !== undefined,
    staleTime: Infinity,
  })
  // Etiqueta sem QR-code não serve: o motorista não consegue bipar o pacote.
  // Por isso só imprime com tudo pronto, e uma falha aparece em vez da caixa vazia.
  const failed = [delivery, me, qr].filter((q) => q.isError)
  const ready = delivery.data && me.data && qr.data ? { delivery: delivery.data, carrier: me.data.carrier.name, qr: qr.data } : null

  return (
    <div className="mx-auto max-w-md px-4 py-6 print:m-0 print:max-w-none print:p-0">
      <div className="mb-4 flex items-center justify-between print:hidden">
        <Link to={`/transportadora/entregas/${id}`} className="font-medium text-brand-700 hover:underline">
          ← Entrega
        </Link>
        {ready && <Button onClick={() => window.print()}>Imprimir</Button>}
      </div>
      {!valid || (delivery.error instanceof ApiError && delivery.error.status === 404) ? (
        <p className="text-lg font-semibold">Entrega não encontrada.</p>
      ) : failed.length > 0 ? (
        <LoadError error={failed[0].error} onRetry={() => failed.forEach((q) => void q.refetch())} />
      ) : ready ? (
        <Label {...ready} />
      ) : (
        <Loading label="Carregando etiqueta" />
      )}
    </div>
  )
}

function Label({ delivery: d, carrier, qr }: { delivery: Delivery; carrier: string; qr: string }) {
  const structured = d.postal_code !== ''
  return (
    <article
      aria-label="Etiqueta"
      className="flex flex-col gap-3 border-2 border-slate-900 bg-white p-4 text-slate-900 print:border-0"
    >
      {/* O id interno não vai na etiqueta: em sequência, ele contaria o volume da transportadora. */}
      <header className="border-b-2 border-slate-900 pb-2">
        <span className="text-lg font-bold">{carrier}</span>
      </header>
      <div className="flex items-center gap-4">
        <img src={qr} alt={`QR-code da entrega ${d.tracking_code}`} className="size-36 shrink-0" />
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
