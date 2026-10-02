import { apiGet } from './api'
import type { Status } from './status'

export type Tracking = {
  tracking_code: string
  status: Status
  recipient_first_name: string
  updated_at: string
  events: { status: Status; created_at: string }[]
}

// Mesmo formato de internal/delivery/tracking.go: "RS" + 10 caracteres sem 0, O, 1 e I.
const codeRe = /^RS[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{10}$/

/** Tira espaços e hífens que o cliente cola junto e passa para maiúsculas. */
export function normalizeCode(input: string): string {
  return input.replace(/[\s-]/g, '').toUpperCase()
}

export function isValidCode(code: string): boolean {
  return codeRe.test(code)
}

export function getTracking(code: string, signal?: AbortSignal): Promise<Tracking> {
  return apiGet<Tracking>(`/public/tracking/${encodeURIComponent(code)}`, { signal })
}
