import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi } from '../test/render'
import { refreshSession, sameUser, type Session } from './session'

describe('refreshSession', () => {
  afterEach(() => {
    // @ts-expect-error a trava de mentira só existe neste teste
    delete navigator.locks
  })

  it('chamadas simultâneas usam uma requisição só', async () => {
    const calls = mockApi({ 'POST /auth/refresh': () => [200, { token: 't', role: 'carrier', expires_in: 900 }] })
    const [a, b] = await Promise.all([refreshSession(), refreshSession()])
    expect(a).toEqual({ token: 't', role: 'carrier' })
    expect(b).toBe(a)
    expect(calls).toHaveLength(1)
    expect(calls[0].credentials).toBe('include')
  })

  it('401 vira "sem sessão"', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, { error: 'invalid session' }] })
    await expect(refreshSession()).resolves.toBeNull()
  })

  it('duas abas não renovam ao mesmo tempo: a segunda espera e usa o cookie novo', async () => {
    // Trava como a do navegador: quem pede depois espera quem pediu antes.
    let last: Promise<unknown> = Promise.resolve()
    const names: string[] = []
    Object.defineProperty(navigator, 'locks', {
      configurable: true,
      value: {
        request: (name: string, fn: () => Promise<unknown>) => {
          names.push(name)
          const run = last.then(fn)
          last = run.catch(() => undefined)
          return run
        },
      },
    })
    const log: string[] = []
    let n = 0
    vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
      const id = ++n
      log.push(`início ${id}`)
      await new Promise((r) => setTimeout(r, 5))
      log.push(`fim ${id}`)
      return Response.json({ token: `t${id}`, role: 'carrier', expires_in: 900 })
    })
    // Cada aba tem a sua cópia do módulo, como no navegador.
    vi.resetModules()
    const otherTab = await import('./session')

    const [a, b] = await Promise.all([refreshSession(), otherTab.refreshSession()])
    expect(log).toEqual(['início 1', 'fim 1', 'início 2', 'fim 2'])
    expect([a?.token, b?.token]).toEqual(['t1', 't2'])
    expect(names).toEqual(['rastreia-refresh', 'rastreia-refresh'])
  })
})

describe('sameUser', () => {
  const jwt = (claims: object) =>
    `eyJhbGciOiJIUzI1NiJ9.${btoa(JSON.stringify(claims)).replace(/=+$/, '').replace(/\+/g, '-').replace(/\//g, '_')}.assinatura`
  const session = (sub: string, role: Session['role'] = 'carrier'): Session => ({ token: jwt({ sub, exp: 1 }), role })

  it('compara o dono do token, não o token em si', () => {
    expect(sameUser(session('7'), { ...session('7'), token: jwt({ sub: '7', exp: 2 }) })).toBe(true)
    expect(sameUser(session('7'), session('8'))).toBe(false)
    expect(sameUser(session('7'), session('7', 'driver'))).toBe(false)
  })
})
