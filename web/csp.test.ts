import { describe, expect, it } from 'vitest'
import { contentSecurityPolicy } from './csp.ts'

describe('contentSecurityPolicy', () => {
  it('libera só a origem da API para conexões', () => {
    const csp = contentSecurityPolicy('https://api.rastreia.dev/v1/')
    expect(csp).toContain("connect-src 'self' https://api.rastreia.dev;")
    expect(csp).toContain("script-src 'self';")
    expect(csp).not.toContain('unsafe-inline')
    expect(csp).toContain("object-src 'none'")
  })
})
