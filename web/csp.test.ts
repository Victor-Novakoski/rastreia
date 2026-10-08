import { describe, expect, it } from 'vitest'
import { contentSecurityPolicy } from './csp.ts'

describe('contentSecurityPolicy', () => {
  it('libera só a API e os serviços de endereço para conexões', () => {
    const csp = contentSecurityPolicy('https://api.rastreia.dev/v1/')
    expect(csp).toContain(
      "connect-src 'self' https://api.rastreia.dev wss://api.rastreia.dev https://viacep.com.br https://nominatim.openstreetmap.org;",
    )
    expect(csp).toContain("img-src 'self' data: https://tile.openstreetmap.org;")
    expect(csp).toContain("script-src 'self';")
    expect(csp).not.toContain('unsafe-inline')
    expect(csp).toContain("object-src 'none'")
  })
})
