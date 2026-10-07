import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { carrierSession, mockApi, renderApp } from '../../test/render'

const qrDataURL = vi.hoisted(() => vi.fn<(text: string) => Promise<string>>())
vi.mock('../../lib/qr', async (original) => ({ ...(await original<typeof import('../../lib/qr')>()), qrDataURL }))

const delivery = {
  id: 41,
  tracking_code: 'RS7K2M9QXA4P',
  recipient_name: 'Maria Souza',
  recipient_email: 'maria@example.com',
  recipient_phone: '',
  address: 'Rua A, 10',
  postal_code: '',
  street: '',
  number: '',
  complement: '',
  district: '',
  city: '',
  state: '',
  address_reference: '',
  latitude: null,
  longitude: null,
  status: 'pending',
  driver_id: null,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T10:00:00Z',
  completed_at: null,
  anonymized_at: null,
}
const me = { id: 1, name: 'Carla', email: 'carla@example.com', role: 'carrier', carrier: { id: 1, name: 'Expresso Sul', document: null } }

function api() {
  return mockApi({
    'POST /auth/refresh': () => [200, carrierSession],
    'GET /me': () => [200, me],
    'GET /deliveries/41': () => [200, delivery],
  })
}

describe('etiqueta', () => {
  it('só oferece Imprimir quando o QR-code está pronto', async () => {
    let finish!: (url: string) => void
    qrDataURL.mockReturnValueOnce(new Promise((resolve) => (finish = resolve)))
    api()
    renderApp('/transportadora/entregas/41/etiqueta')
    // A entrega já chegou (o QR-code só é pedido depois dela), mas o QR-code não.
    await waitFor(() => expect(qrDataURL).toHaveBeenCalled())
    expect(screen.getByText('Carregando etiqueta')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Imprimir' })).not.toBeInTheDocument()

    finish('data:image/svg+xml,qr')
    expect(await screen.findByRole('img', { name: 'QR-code da entrega RS7K2M9QXA4P' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Imprimir' })).toBeInTheDocument()
    expect(qrDataURL).toHaveBeenCalledWith(`${window.location.origin}/rastreio/RS7K2M9QXA4P`)
    // O id interno não sai na etiqueta.
    expect(screen.queryByText(/41/)).not.toBeInTheDocument()
  })

  it('QR-code que não foi gerado mostra o erro, sem Imprimir, e deixa tentar de novo', async () => {
    qrDataURL.mockRejectedValueOnce(new Error('chunk failed')).mockResolvedValueOnce('data:image/svg+xml,qr')
    api()
    renderApp('/transportadora/entregas/41/etiqueta')
    expect(await screen.findByText('Não foi possível carregar')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Imprimir' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tentar de novo' }))
    expect(await screen.findByRole('button', { name: 'Imprimir' })).toBeInTheDocument()
    expect(screen.getByText('Expresso Sul')).toBeInTheDocument()
  })

  it('sem os dados da transportadora também não imprime', async () => {
    qrDataURL.mockResolvedValue('data:image/svg+xml,qr')
    mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /me': () => [500, { error: 'internal error' }],
      'GET /deliveries/41': () => [200, delivery],
    })
    renderApp('/transportadora/entregas/41/etiqueta')
    expect(await screen.findByText('Algo deu errado do nosso lado.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Imprimir' })).not.toBeInTheDocument()
  })
})
