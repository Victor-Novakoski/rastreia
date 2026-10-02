import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { mockApi, renderApp } from '../../test/render'

const session = { token: 't', role: 'driver', expires_in: 900 }
const base = {
  recipient_email: 'x@example.com',
  driver_id: 2,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T12:00:00Z',
  completed_at: null,
  anonymized_at: null,
}
const pickedUp = { ...base, id: 1, tracking_code: 'RS7K2M9QXA4P', recipient_name: 'Maria', address: 'Rua A, 10', status: 'picked_up' }
const delivered = { ...base, id: 2, tracking_code: 'RSQ8W3ZK5MNB', recipient_name: 'Carlos', address: 'Rua B, 20', status: 'delivered' }

describe('app do motorista', () => {
  it('lista pendentes primeiro e guarda as entregues', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, session], 'GET /me/deliveries': () => [200, [delivered, pickedUp]] })
    renderApp('/motorista')
    expect(await screen.findByRole('heading', { name: 'Para fazer (1)' })).toBeInTheDocument()
    expect(screen.getByText('Entregues (1)')).toBeInTheDocument()
  })

  it('um toque muda o status', async () => {
    let current = pickedUp
    const calls = mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/deliveries': () => [200, [current]],
      'GET /deliveries/1/events': () => [200, []],
      'POST /deliveries/1/events': ({ body }) => {
        current = { ...current, status: (body as { status: string }).status }
        return [201, { id: 9, delivery_id: 1, status: current.status, note: null, created_by: 2, created_at: '2026-10-01T13:00:00Z' }]
      },
    })
    renderApp('/motorista/entregas/1')
    await userEvent.click(await screen.findByRole('button', { name: 'Saí para entrega' }))
    expect(await screen.findByText('Status atualizado: Em rota.')).toBeInTheDocument()
    expect(calls.find((c) => c.method === 'POST' && c.path === '/deliveries/1/events')!.body).toEqual({ status: 'in_transit' })
    expect(await screen.findByRole('button', { name: 'Entreguei' })).toBeInTheDocument()
  })

  it('falha pede o motivo', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/deliveries': () => [200, [pickedUp]],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/motorista/entregas/1')
    await userEvent.click(await screen.findByRole('button', { name: 'Não consegui entregar' }))
    expect(screen.getByLabelText('O que aconteceu?')).toBeRequired()
  })

  it('409 de uma repetição que já tinha chegado conta como sucesso', async () => {
    let listCalls = 0
    mockApi({
      'POST /auth/refresh': () => [200, session],
      // A segunda leitura já mostra o status novo.
      'GET /me/deliveries': () => [200, [listCalls++ === 0 ? pickedUp : { ...pickedUp, status: 'in_transit' }]],
      'GET /deliveries/1/events': () => [200, []],
      'POST /deliveries/1/events': () => [409, { error: 'status changed' }],
    })
    renderApp('/motorista/entregas/1')
    await userEvent.click(await screen.findByRole('button', { name: 'Saí para entrega' }))
    expect(await screen.findByText('Status atualizado: Em rota.')).toBeInTheDocument()
  })

  it('sem sinal mostra o aviso e não perde a tela', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/deliveries': () => [200, [pickedUp]],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/motorista/entregas/1')
    const button = await screen.findByRole('button', { name: 'Saí para entrega' })
    const { vi } = await import('vitest')
    vi.mocked(globalThis.fetch).mockRejectedValueOnce(new TypeError('Failed to fetch'))
    await userEvent.click(button)
    expect(await screen.findByText('Sem sinal. Quando voltar, toque de novo.')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Saí para entrega' })).toBeEnabled())
  })
})
