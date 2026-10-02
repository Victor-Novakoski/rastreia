import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { FakeWebSocket } from '../test/websocket'
import { connectLive, UNAUTHORIZED } from './live'

const flush = () => vi.advanceTimersByTimeAsync(0)

describe('connectLive', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('troca http por ws e entrega as mensagens já convertidas', async () => {
    const onMessage = vi.fn()
    connectLive('/public/tracking/RS7K2M9QXA4P/live', { onMessage })
    await flush()
    const ws = FakeWebSocket.instances[0]
    expect(ws.url).toBe('ws://localhost:8080/public/tracking/RS7K2M9QXA4P/live')
    ws.open()
    ws.receive({ status: 'picked_up' })
    expect(onMessage).toHaveBeenCalledWith({ status: 'picked_up' })
    expect(ws.sent).toEqual([])
  })

  it('manda o token na primeira mensagem, nunca na URL', async () => {
    connectLive('/live/deliveries', { onMessage: () => {}, token: async () => 'tok' })
    await flush()
    const ws = FakeWebSocket.instances[0]
    ws.open()
    expect(ws.url).not.toContain('tok')
    expect(ws.sent).toEqual([JSON.stringify({ token: 'tok' })])
  })

  it('reconecta esperando cada vez mais e avisa que houve queda', async () => {
    const onOpen = vi.fn()
    connectLive('/x', { onMessage: () => {}, onOpen })
    await flush()
    FakeWebSocket.instances[0].open()
    expect(onOpen).toHaveBeenLastCalledWith(false)

    FakeWebSocket.instances[0].drop()
    await vi.advanceTimersByTimeAsync(999)
    expect(FakeWebSocket.instances).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(FakeWebSocket.instances).toHaveLength(2)

    FakeWebSocket.instances[1].drop()
    await vi.advanceTimersByTimeAsync(1999)
    expect(FakeWebSocket.instances).toHaveLength(2)
    await vi.advanceTimersByTimeAsync(1)
    FakeWebSocket.instances[2].open()
    expect(onOpen).toHaveBeenLastCalledWith(true)
  })

  it('renova o token quando a API recusa o atual', async () => {
    const token = vi.fn(async (renew: boolean) => (renew ? 'novo' : 'velho'))
    connectLive('/live/deliveries', { onMessage: () => {}, token })
    await flush()
    FakeWebSocket.instances[0].open()
    FakeWebSocket.instances[0].drop(UNAUTHORIZED)
    await vi.advanceTimersByTimeAsync(1000)
    expect(token).toHaveBeenLastCalledWith(true)
    FakeWebSocket.instances[1].open()
    expect(FakeWebSocket.instances[1].sent).toEqual([JSON.stringify({ token: 'novo' })])
  })

  it('sem sessão não conecta', async () => {
    connectLive('/live/deliveries', { onMessage: () => {}, token: async () => null })
    await flush()
    expect(FakeWebSocket.instances).toHaveLength(0)
  })

  it('fechar encerra a conexão e para de reconectar', async () => {
    const close = connectLive('/x', { onMessage: () => {} })
    await flush()
    const ws = FakeWebSocket.instances[0]
    close()
    expect(ws.closed).toBe(true)
    ws.drop()
    await vi.advanceTimersByTimeAsync(60_000)
    expect(FakeWebSocket.instances).toHaveLength(1)
  })
})
