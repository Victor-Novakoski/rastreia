import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { Alert } from '../../components/Alert'
import { Button } from '../../components/Button'
import { TextArea } from '../../components/Field'
import { StatusBadge } from '../../components/StatusBadge'
import { LoadError, Loading } from '../../components/States'
import { ApiError, errorMessage } from '../../lib/api'
import { useAuth } from '../../lib/auth'
import { addEvent, getDelivery, listEvents, type Delivery } from '../../lib/deliveries'
import { actionLabel, retryLabel } from '../../lib/driver'
import { routeKey } from '../../lib/route'
import { fieldErrors } from '../../lib/fields'
import { formatPhone } from '../../lib/address'
import { formatDateTime } from '../../lib/format'
import { nextStatuses, statusInfo, type Status } from '../../lib/status'

export function DriverDeliveryPage() {
  const id = Number(useParams().id)
  const { api } = useAuth()
  const queryClient = useQueryClient()
  const valid = Number.isInteger(id) && id > 0
  const delivery = useQuery({
    queryKey: ['me', 'deliveries', id],
    queryFn: ({ signal }) => getDelivery(api, id, signal),
    enabled: valid,
    // Vinda da lista, a entrega aparece na hora enquanto a versão nova chega.
    placeholderData: () => queryClient.getQueryData<Delivery[]>(['me', 'deliveries'])?.find((d) => d.id === id),
    // Na rua o sinal cai: tenta de novo algumas vezes, mas não quando ela não existe.
    retry: (count, err) => !(err instanceof ApiError && err.status === 404) && count < 3,
  })

  return (
    <>
      <Link to="/motorista" className="inline-flex min-h-11 items-center font-medium text-brand-700">
        ← Minhas entregas
      </Link>
      {!valid || (delivery.error instanceof ApiError && delivery.error.status === 404) ? (
        <p className="mt-4 text-lg font-semibold">Entrega não encontrada.</p>
      ) : delivery.data ? (
        <Details delivery={delivery.data} />
      ) : delivery.isError ? (
        <LoadError error={delivery.error} onRetry={() => void delivery.refetch()} />
      ) : (
        <Loading label="Carregando entrega" />
      )}
    </>
  )
}

function Details({ delivery: d }: { delivery: Delivery }) {
  const place = d.latitude != null && d.longitude != null ? `${d.latitude},${d.longitude}` : d.address
  const maps = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(place)}`
  return (
    <div className="flex flex-col gap-6">
      <section className="rounded-xl border border-slate-200 bg-white p-4">
        <StatusBadge status={d.status} size="lg" />
        <h1 className="mt-3 text-2xl font-bold">{d.recipient_name}</h1>
        <p className="mt-1 text-lg">{d.address}</p>
        {d.address_reference && <p className="mt-1 text-slate-700">Referência: {d.address_reference}</p>}
        {d.recipient_phone && (
          <a
            href={`tel:+55${d.recipient_phone}`}
            className="mt-3 mr-3 inline-flex min-h-11 items-center rounded-lg border border-slate-300 px-4 font-semibold text-brand-700"
          >
            Ligar {formatPhone(d.recipient_phone)}
          </a>
        )}
        <a
          href={maps}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-3 inline-flex min-h-11 items-center rounded-lg border border-slate-300 px-4 font-semibold text-brand-700"
        >
          Abrir no mapa
        </a>
        <p className="mt-3 font-mono text-sm text-slate-600">{d.tracking_code}</p>
      </section>

      <Actions delivery={d} />

      <section>
        <h2 className="text-lg font-semibold">Histórico</h2>
        <History id={d.id} />
      </section>
    </div>
  )
}

function Actions({ delivery: d }: { delivery: Delivery }) {
  const { api } = useAuth()
  const queryClient = useQueryClient()
  const [failing, setFailing] = useState(false)
  const [note, setNote] = useState('')
  const [done, setDone] = useState('')

  const change = useMutation({
    mutationFn: async (input: { status: Status; note?: string }) => {
      try {
        await addEvent(api, d.id, input)
      } catch (err) {
        // Com sinal ruim, a primeira tentativa pode ter chegado e a resposta
        // não: a repetição recebe 409. Se o status já é o pedido, deu certo.
        if (err instanceof ApiError && err.status === 409) {
          const fresh = await getDelivery(api, d.id)
          if (fresh.status === input.status) return
        }
        throw err
      }
    },
    onMutate: () => setDone(''),
    onSuccess: (_, input) => {
      setDone(`Status atualizado: ${statusInfo[input.status].label}.`)
      setFailing(false)
      setNote('')
    },
    onSettled: () => {
      // A lista, esta entrega e a rota mostram o status.
      void queryClient.invalidateQueries({ queryKey: ['me', 'deliveries'] })
      void queryClient.invalidateQueries({ queryKey: routeKey })
      void queryClient.invalidateQueries({ queryKey: ['events', d.id] })
    },
  })

  const options = nextStatuses[d.status]
  if (options.length === 0) {
    return done ? <Alert tone="success">{done}</Alert> : null
  }

  const errors = fieldErrors(change.error)
  const message =
    change.error instanceof ApiError && change.error.status === 0
      ? 'Sem sinal. Quando voltar, toque de novo.'
      : change.error instanceof ApiError && change.error.status === 409
        ? 'O status desta entrega mudou. Confira abaixo.'
        : errorMessage(change.error)

  return (
    <section className="flex flex-col gap-3" aria-label="Atualizar status">
      {done && <Alert tone="success">{done}</Alert>}
      {change.isError && !errors.note && <Alert>{message}</Alert>}
      {failing ? (
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (!change.isPending) change.mutate({ status: 'failed', note: note.trim() })
          }}
          className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4"
        >
          <TextArea
            label="O que aconteceu?"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            required
            maxLength={500}
            rows={3}
            error={errors.note}
            hint="Ex.: ninguém em casa, endereço não encontrado."
          />
          <Button type="submit" loading={change.isPending} variant="danger" className="min-h-14 text-lg">
            Registrar falha
          </Button>
          <button type="button" onClick={() => setFailing(false)} className="min-h-11 font-semibold text-slate-700">
            Voltar
          </button>
        </form>
      ) : (
        options.map((s) =>
          s === 'failed' ? (
            <button
              key={s}
              type="button"
              disabled={change.isPending}
              onClick={() => setFailing(true)}
              className="min-h-14 rounded-xl border-2 border-danger-fg bg-white text-lg font-semibold text-danger-fg disabled:opacity-60"
            >
              {actionLabel.failed}
            </button>
          ) : (
            <Button
              key={s}
              type="button"
              loading={change.isPending && change.variables?.status === s}
              disabled={change.isPending}
              onClick={() => change.mutate({ status: s })}
              className="min-h-14 text-lg"
            >
              {d.status === 'failed' ? retryLabel : actionLabel[s]}
            </Button>
          ),
        )
      )}
    </section>
  )
}

function History({ id }: { id: number }) {
  const { api } = useAuth()
  const events = useQuery({ queryKey: ['events', id], queryFn: ({ signal }) => listEvents(api, id, signal) })
  if (events.isPending) return <Loading label="Carregando histórico" />
  if (events.isError) return <LoadError error={events.error} onRetry={() => void events.refetch()} />
  return (
    <ol className="mt-2 flex flex-col gap-3">
      {[...events.data].reverse().map((ev) => (
        <li key={ev.id} className="rounded-lg bg-white p-3">
          <p className="font-medium">{statusInfo[ev.status].label}</p>
          <p className="text-sm text-slate-600">{formatDateTime(ev.created_at)}</p>
          {ev.note && <p className="mt-1 text-sm whitespace-pre-line">{ev.note}</p>}
        </li>
      ))}
    </ol>
  )
}
