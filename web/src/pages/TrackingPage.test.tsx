import { act, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import { FakeWebSocket } from '../test/websocket'
import { TrackingPage } from './TrackingPage'

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rastreio/:code" element={<TrackingPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

/**
 * O WebSocket abre num efeito, depois que a tela já mostrou a entrega: com a
 * máquina ocupada, o teste chega antes dele. Espera a conexão existir.
 */
function liveSocket() {
  return vi.waitFor(() => {
    const ws = FakeWebSocket.last('/public/tracking/RS7K2M9QXA4P/live')
    if (!ws) throw new Error('o WebSocket ainda não abriu')
    return ws
  })
}

function mockFetch(status: number, body: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }),
  )
}

describe('TrackingPage', () => {
  it('mostra status, nome e histórico do mais recente para o mais antigo', async () => {
    const fetch = mockFetch(200, {
      tracking_code: 'RS7K2M9QXA4P',
      status: 'in_transit',
      recipient_first_name: 'Maria',
      updated_at: '2026-10-01T15:00:00Z',
      events: [
        { status: 'pending', created_at: '2026-10-01T10:00:00Z' },
        { status: 'picked_up', created_at: '2026-10-01T12:00:00Z' },
        { status: 'in_transit', created_at: '2026-10-01T15:00:00Z' },
      ],
    })
    renderAt('/rastreio/rs7k2m9qxa4p')

    expect(await screen.findByRole('heading', { name: 'Olá, Maria' })).toBeInTheDocument()
    expect(fetch).toHaveBeenCalledWith(expect.stringMatching(/\/public\/tracking\/RS7K2M9QXA4P$/), expect.anything())
    const items = screen.getAllByRole('listitem').map((li) => li.textContent)
    expect(items[0]).toContain('Em rota')
    expect(items[2]).toContain('Aguardando coleta')
  })

  it('404 vira "não encontramos"', async () => {
    mockFetch(404, { error: 'not found' })
    renderAt('/rastreio/RS7K2M9QXA4P')
    expect(await screen.findByText('Não encontramos essa entrega')).toBeInTheDocument()
  })

  it('código malformado não chama a API', async () => {
    const fetch = mockFetch(200, {})
    renderAt('/rastreio/abc')
    expect(await screen.findByText('Não encontramos essa entrega')).toBeInTheDocument()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('429 pede para esperar e oferece tentar de novo', async () => {
    mockFetch(429, { error: 'too many requests' })
    renderAt('/rastreio/RS7K2M9QXA4P')
    expect(await screen.findByText(/Muitas consultas seguidas/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tentar de novo' })).toBeInTheDocument()
  })

  it('atualiza sozinho pelo WebSocket depois de carregar', async () => {
    const tracking = {
      tracking_code: 'RS7K2M9QXA4P',
      status: 'in_transit',
      recipient_first_name: 'Maria',
      updated_at: '2026-10-01T15:00:00Z',
      events: [{ status: 'in_transit', created_at: '2026-10-01T15:00:00Z' }],
    }
    mockFetch(200, tracking)
    renderAt('/rastreio/RS7K2M9QXA4P')
    expect(await screen.findByRole('heading', { name: 'Olá, Maria' })).toBeInTheDocument()

    const ws = await liveSocket()
    act(() => ws.open())
    expect(screen.getByRole('status')).toHaveTextContent('Ao vivo')

    act(() =>
      ws.receive({
        ...tracking,
        status: 'delivered',
        updated_at: '2026-10-01T18:00:00Z',
        events: [...tracking.events, { status: 'delivered', created_at: '2026-10-01T18:00:00Z' }],
      }),
    )
    expect(screen.getAllByRole('listitem')[0]).toHaveTextContent('Entregue')

    act(() => ws.drop())
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('depois de uma queda, busca de novo o que mudou enquanto estava sem conexão', async () => {
    const tracking = {
      tracking_code: 'RS7K2M9QXA4P',
      status: 'in_transit',
      recipient_first_name: 'Maria',
      updated_at: '2026-10-01T15:00:00Z',
      events: [{ status: 'in_transit', created_at: '2026-10-01T15:00:00Z' }],
    }
    const fetch = mockFetch(200, tracking)
    renderAt('/rastreio/RS7K2M9QXA4P')
    expect(await screen.findByRole('heading', { name: 'Olá, Maria' })).toBeInTheDocument()
    const ws = await liveSocket()
    act(() => ws.open())

    // Caiu; enquanto isso a entrega foi feita e o aviso se perdeu.
    act(() => ws.drop())
    fetch.mockResolvedValue(
      Response.json({
        ...tracking,
        status: 'delivered',
        updated_at: '2026-10-01T18:00:00Z',
        events: [...tracking.events, { status: 'delivered', created_at: '2026-10-01T18:00:00Z' }],
      }),
    )
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(2), { timeout: 3000 })
    act(() => FakeWebSocket.instances[1].open())
    await vi.waitFor(() => expect(screen.getAllByRole('listitem')[0]).toHaveTextContent('Entregue'))
    expect(fetch).toHaveBeenCalledTimes(2)
  })

  it('código inexistente não abre WebSocket', async () => {
    mockFetch(404, { error: 'not found' })
    renderAt('/rastreio/RS7K2M9QXA4P')
    expect(await screen.findByText('Não encontramos essa entrega')).toBeInTheDocument()
    expect(FakeWebSocket.instances).toHaveLength(0)
  })
})
