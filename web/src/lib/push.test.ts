import { afterEach, describe, expect, it, vi } from 'vitest'
import { disablePush, enablePush, keyBytes, pushState } from './push'

type FakeSub = { endpoint: string; toJSON: () => unknown; unsubscribe: () => Promise<boolean> }

function fakeBrowser({ permission = 'default', subscribed = false } = {}) {
  let sub: FakeSub | null = null
  const makeSub = (): FakeSub => ({
    endpoint: 'https://fcm.googleapis.com/fcm/send/abc',
    toJSON: () => ({ endpoint: 'https://fcm.googleapis.com/fcm/send/abc', keys: { p256dh: 'p', auth: 'a' } }),
    unsubscribe: vi.fn(async () => {
      sub = null
      return true
    }),
  })
  if (subscribed) sub = makeSub()
  const pushManager = {
    getSubscription: vi.fn(async () => sub),
    subscribe: vi.fn(async () => (sub = makeSub())),
  }
  const reg = { pushManager }
  vi.stubGlobal('Notification', { permission, requestPermission: vi.fn(async () => 'granted') })
  vi.stubGlobal('PushManager', function PushManager() {})
  Object.defineProperty(navigator, 'serviceWorker', {
    configurable: true,
    value: { register: vi.fn(async () => reg), ready: Promise.resolve(reg), getRegistration: vi.fn(async () => reg) },
  })
  return { pushManager, getSub: () => sub }
}

function mockFetch() {
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input)
    if (url.endsWith('/public/push/key')) return Response.json({ public_key: 'AQID' })
    return new Response(null, { status: 204 })
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  // @ts-expect-error: remove o serviceWorker falso
  delete navigator.serviceWorker
  localStorage.clear()
})

describe('keyBytes', () => {
  it('decodifica base64url sem padding', () => {
    expect([...keyBytes('AQID')]).toEqual([1, 2, 3])
    expect([...keyBytes('-_8')]).toEqual([251, 255])
  })
})

describe('pushState', () => {
  it('sem suporte no jsdom', async () => {
    expect(await pushState('RS7K2M9QXA4P')).toBe('unsupported')
  })

  it('iPhone fora da tela de início pede para instalar', async () => {
    vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)')
    expect(await pushState('RS7K2M9QXA4P')).toBe('ios-install')
  })

  it('bloqueado pelo usuário', async () => {
    fakeBrowser({ permission: 'denied' })
    expect(await pushState('RS7K2M9QXA4P')).toBe('denied')
  })
})

describe('enablePush e disablePush', () => {
  it('inscreve, registra na API e lembra do código', async () => {
    const { pushManager } = fakeBrowser()
    const fetch = mockFetch()

    expect(await enablePush('RS7K2M9QXA4P')).toBe('on')
    expect(pushManager.subscribe).toHaveBeenCalledWith({
      userVisibleOnly: true,
      applicationServerKey: new Uint8Array([1, 2, 3]),
    })
    const [url, init] = fetch.mock.calls[1]
    expect(String(url)).toMatch(/\/public\/tracking\/RS7K2M9QXA4P\/push$/)
    expect(init?.method).toBe('POST')
    expect(await pushState('RS7K2M9QXA4P')).toBe('on')
    expect(await pushState('RSOUTROCODE2')).toBe('off')
  })

  it('desativar avisa a API e cancela a inscrição quando não segue mais nada', async () => {
    const browser = fakeBrowser({ subscribed: true })
    const fetch = mockFetch()
    await enablePush('RS7K2M9QXA4P')

    expect(await disablePush('RS7K2M9QXA4P')).toBe('off')
    const last = fetch.mock.calls.at(-1)!
    expect(last[1]?.method).toBe('DELETE')
    expect(JSON.parse(String(last[1]?.body))).toEqual({ endpoint: 'https://fcm.googleapis.com/fcm/send/abc' })
    expect(browser.getSub()).toBeNull()
  })
})
