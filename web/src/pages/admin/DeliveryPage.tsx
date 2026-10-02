import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useLocation, useParams } from 'react-router'
import { Alert } from '../../components/Alert'
import { DeliveryForm } from '../../components/DeliveryForm'
import { EventForm } from '../../components/EventForm'
import { StatusBadge } from '../../components/StatusBadge'
import { StatusIcon } from '../../components/StatusIcon'
import { LoadError, Loading } from '../../components/States'
import { ApiError, errorMessage } from '../../lib/api'
import { useAuth } from '../../lib/auth'
import {
  addEvent,
  getDelivery,
  listEvents,
  updateDelivery,
  type Delivery,
  type DeliveryInput,
} from '../../lib/deliveries'
import { fieldErrors } from '../../lib/fields'
import { formatDateTime } from '../../lib/format'
import { useDrivers } from '../../lib/queries'
import { statusInfo, toneClasses, type Status } from '../../lib/status'

export function DeliveryPage() {
  const id = Number(useParams().id)
  const { api } = useAuth()
  const delivery = useQuery({
    queryKey: ['delivery', id],
    queryFn: ({ signal }) => getDelivery(api, id, signal),
    enabled: Number.isInteger(id) && id > 0,
  })

  return (
    <div className="max-w-4xl">
      <Link to="/admin/entregas" className="text-sm font-medium text-brand-700 hover:underline">
        ← Entregas
      </Link>
      {!Number.isInteger(id) || id <= 0 || (delivery.error instanceof ApiError && delivery.error.status === 404) ? (
        <p className="mt-4 text-lg font-semibold">Entrega não encontrada.</p>
      ) : delivery.isPending ? (
        <Loading label="Carregando entrega" />
      ) : delivery.isError ? (
        <LoadError error={delivery.error} onRetry={() => void delivery.refetch()} />
      ) : (
        <Details delivery={delivery.data} />
      )}
    </div>
  )
}

function Details({ delivery }: { delivery: Delivery }) {
  const created = (useLocation().state as { created?: boolean } | null)?.created
  return (
    <>
      <div className="mt-2 flex flex-wrap items-center gap-3">
        <h1 className="font-mono text-2xl font-bold tracking-wider">{delivery.tracking_code}</h1>
        <StatusBadge status={delivery.status} />
      </div>
      <p className="mt-1 text-sm text-slate-600">
        Criada em {formatDateTime(delivery.created_at)} ·{' '}
        <Link to={`/rastreio/${delivery.tracking_code}`} className="text-brand-700 underline underline-offset-2">
          ver rastreio público
        </Link>
      </p>
      {delivery.anonymized_at && (
        <div className="mt-4">
          <Alert tone="success">
            Os dados do destinatário foram apagados em {formatDateTime(delivery.anonymized_at)}, como manda a política
            de retenção. O histórico fica, mas a entrega não pode mais ser alterada.
          </Alert>
        </div>
      )}
      {created && (
        <div className="mt-4">
          <Alert tone="success">Entrega criada.</Alert>
        </div>
      )}

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <section className="rounded-lg border border-slate-200 bg-white p-4">
          {!delivery.anonymized_at && (
            <>
              <h2 className="text-lg font-semibold">Mudar status</h2>
              <div className="mt-3">
                <StatusChange delivery={delivery} />
              </div>
            </>
          )}
          <h2 className={delivery.anonymized_at ? 'text-lg font-semibold' : 'mt-6 text-lg font-semibold'}>Histórico</h2>
          <History id={delivery.id} />
        </section>
        <section className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-lg font-semibold">Dados da entrega</h2>
          <div className="mt-3">
            {delivery.anonymized_at ? (
              <p className="text-slate-600">Dados apagados.</p>
            ) : (
              <EditDelivery delivery={delivery} />
            )}
          </div>
        </section>
      </div>
    </>
  )
}

function StatusChange({ delivery }: { delivery: Delivery }) {
  const { api } = useAuth()
  const queryClient = useQueryClient()
  const change = useMutation({
    mutationFn: (input: { status: Status; note?: string }) => addEvent(api, delivery.id, input),
    onSettled: () => {
      // Também depois de um 409: alguém pode ter mudado o status antes.
      void queryClient.invalidateQueries({ queryKey: ['delivery', delivery.id] })
      void queryClient.invalidateQueries({ queryKey: ['events', delivery.id] })
      void queryClient.invalidateQueries({ queryKey: ['deliveries'] })
    },
  })
  const conflict = change.error instanceof ApiError && change.error.status === 409
  return (
    <div className="flex flex-col gap-3">
      {change.isError && (
        <Alert>{conflict ? 'O status mudou enquanto você editava. Confira o histórico.' : errorMessage(change.error)}</Alert>
      )}
      <EventForm
        // Remonta quando o status muda, limpando o formulário.
        key={delivery.status}
        current={delivery.status}
        errors={fieldErrors(change.error)}
        sending={change.isPending}
        onSubmit={(input) => change.mutate(input)}
      />
    </div>
  )
}

function History({ id }: { id: number }) {
  const { api } = useAuth()
  const events = useQuery({ queryKey: ['events', id], queryFn: ({ signal }) => listEvents(api, id, signal) })
  if (events.isPending) return <Loading label="Carregando histórico" />
  if (events.isError) return <LoadError error={events.error} onRetry={() => void events.refetch()} />
  const list = [...events.data].reverse()
  return (
    <ol className="mt-3 flex flex-col gap-4">
      {list.map((ev) => (
        <li key={ev.id} className="flex gap-3">
          <span
            className={`flex size-8 shrink-0 items-center justify-center rounded-full ${toneClasses[statusInfo[ev.status].tone]}`}
          >
            <StatusIcon status={ev.status} className="size-4" />
          </span>
          <div>
            <p className="font-medium">{statusInfo[ev.status].label}</p>
            <p className="text-sm text-slate-600">{formatDateTime(ev.created_at)}</p>
            {ev.note && <p className="mt-1 text-sm whitespace-pre-line">{ev.note}</p>}
          </div>
        </li>
      ))}
    </ol>
  )
}

function EditDelivery({ delivery }: { delivery: Delivery }) {
  const { api } = useAuth()
  const queryClient = useQueryClient()
  const drivers = useDrivers()
  const [saved, setSaved] = useState(false)
  const save = useMutation({
    mutationFn: (input: DeliveryInput) => updateDelivery(api, delivery.id, input),
    onMutate: () => setSaved(false),
    onSuccess: (d) => {
      queryClient.setQueryData(['delivery', d.id], d)
      void queryClient.invalidateQueries({ queryKey: ['deliveries'] })
      setSaved(true)
    },
  })

  if (drivers.isPending) return <Loading />
  if (drivers.isError) return <LoadError error={drivers.error} onRetry={() => void drivers.refetch()} />
  return (
    <div className="flex flex-col gap-3">
      {saved && <Alert tone="success">Alterações salvas.</Alert>}
      {save.isError && <Alert>{errorMessage(save.error)}</Alert>}
      <DeliveryForm
        initial={{
          recipient_name: delivery.recipient_name,
          recipient_email: delivery.recipient_email,
          address: delivery.address,
          driver_id: delivery.driver_id ?? undefined,
        }}
        drivers={drivers.data}
        errors={fieldErrors(save.error)}
        sending={save.isPending}
        submitLabel="Salvar"
        onSubmit={(input) => save.mutate(input)}
      />
    </div>
  )
}
