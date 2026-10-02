import type { AuthContextValue } from './auth'
import type { Status } from './status'

type Api = AuthContextValue['api']

export type Delivery = {
  id: number
  tracking_code: string
  recipient_name: string
  recipient_email: string
  address: string
  status: Status
  driver_id: number | null
  created_at: string
  updated_at: string
  completed_at: string | null
}

export type DeliveryEvent = {
  id: number
  delivery_id: number
  status: Status
  note: string | null
  created_by: number | null
  created_at: string
}

export type Driver = { id: number; name: string; email: string; role: 'driver'; created_at: string }

export type DeliveryInput = {
  recipient_name: string
  recipient_email: string
  address: string
  driver_id?: number
}

export const pageSize = 20

export function listDeliveries(api: Api, params: { status?: Status; page: number }, signal?: AbortSignal) {
  const q = new URLSearchParams({ page: String(params.page), size: String(pageSize) })
  if (params.status) q.set('status', params.status)
  return api<Delivery[]>(`/deliveries?${q}`, { signal })
}

/** Até 100 entregas do motorista, mais recentes primeiro. */
export function listMyDeliveries(api: Api, signal?: AbortSignal) {
  return api<Delivery[]>('/me/deliveries?size=100', { signal })
}

export function getDelivery(api: Api, id: number, signal?: AbortSignal) {
  return api<Delivery>(`/deliveries/${id}`, { signal })
}

/** A chave evita criar a entrega duas vezes se o envio for repetido (SECURITY.md #9). */
export function createDelivery(api: Api, input: DeliveryInput, idempotencyKey: string) {
  return api<Delivery>('/deliveries', {
    method: 'POST',
    body: input,
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}

export function updateDelivery(api: Api, id: number, input: Partial<DeliveryInput>) {
  return api<Delivery>(`/deliveries/${id}`, { method: 'PATCH', body: input })
}

export function listEvents(api: Api, id: number, signal?: AbortSignal) {
  return api<DeliveryEvent[]>(`/deliveries/${id}/events`, { signal })
}

export function addEvent(api: Api, id: number, input: { status: Status; note?: string }) {
  return api<DeliveryEvent>(`/deliveries/${id}/events`, { method: 'POST', body: input })
}

export function listDrivers(api: Api, signal?: AbortSignal) {
  return api<Driver[]>('/drivers', { signal })
}

export function createDriver(api: Api, input: { name: string; email: string; password: string }) {
  return api<Driver>('/drivers', { method: 'POST', body: input })
}
