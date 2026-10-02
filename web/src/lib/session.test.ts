import { describe, expect, it } from 'vitest'
import { mockApi } from '../test/render'
import { refreshSession } from './session'

describe('refreshSession', () => {
  it('chamadas simultâneas usam uma requisição só', async () => {
    const calls = mockApi({ 'POST /auth/refresh': () => [200, { token: 't', role: 'admin', expires_in: 900 }] })
    const [a, b] = await Promise.all([refreshSession(), refreshSession()])
    expect(a).toEqual({ token: 't', role: 'admin' })
    expect(b).toBe(a)
    expect(calls).toHaveLength(1)
    expect(calls[0].credentials).toBe('include')
  })

  it('401 vira "sem sessão"', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, { error: 'invalid session' }] })
    await expect(refreshSession()).resolves.toBeNull()
  })
})
