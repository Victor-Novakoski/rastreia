import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { QrScanner } from './QrScanner'

// Sem BarcodeDetector (iPhone, Firefox) o leitor é o jsQR, baixado na hora.
// Aqui o download falha, como sem sinal.
vi.mock('jsqr', () => {
  throw new TypeError('Failed to fetch dynamically imported module')
})

function fakeCamera(getUserMedia: () => Promise<MediaStream>) {
  Object.defineProperty(navigator, 'mediaDevices', { configurable: true, value: { getUserMedia } })
}

function fakeStream() {
  const stop = vi.fn()
  return { stream: { getTracks: () => [{ stop }] } as unknown as MediaStream, stop }
}

describe('QrScanner', () => {
  beforeEach(() => {
    vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined)
  })
  afterEach(() => {
    // @ts-expect-error a câmera de mentira só existe nestes testes
    delete navigator.mediaDevices
  })

  it('fechar enquanto a câmera liga desliga a câmera quando ela chegar', async () => {
    let grant!: (s: MediaStream) => void
    fakeCamera(() => new Promise((resolve) => (grant = resolve)))
    const { unmount } = render(<QrScanner onCode={() => {}} onClose={() => {}} />)
    unmount()

    const { stream, stop } = fakeStream()
    grant(stream)
    await vi.waitFor(() => expect(stop).toHaveBeenCalled())
  })

  it('leitor que não baixou desliga a câmera e pede para digitar o código', async () => {
    const { stream, stop } = fakeStream()
    fakeCamera(async () => stream)
    render(<QrScanner onCode={() => {}} onClose={() => {}} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Não deu para ligar o leitor de QR-code. Digite o código.')
    expect(stop).toHaveBeenCalled()
  })

  it('sem permissão para a câmera, explica e oferece digitar', async () => {
    fakeCamera(async () => {
      throw new DOMException('denied', 'NotAllowedError')
    })
    render(<QrScanner onCode={() => {}} onClose={() => {}} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Sem acesso à câmera')
  })
})
