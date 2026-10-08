import { useEffect, useRef, useState } from 'react'

type Props = {
  /** Chamado a cada código lido; o mesmo código só repete depois de alguns segundos. */
  onCode: (text: string) => void
  onClose: () => void
}

type Detector = { detect: (source: HTMLVideoElement) => Promise<{ rawValue: string }[]> }
type DetectorClass = new (opts: { formats: string[] }) => Detector

/**
 * Leitor de QR-code pela câmera traseira, para bipar um pacote atrás do
 * outro. Usa o leitor do navegador (BarcodeDetector) quando existe, como no
 * Chrome do Android; nos outros, o jsQR lê os quadros do vídeo.
 */
export function QrScanner({ onCode, onClose }: Props) {
  const video = useRef<HTMLVideoElement>(null)
  const [error, setError] = useState('')
  const handler = useRef(onCode)
  useEffect(() => {
    handler.current = onCode
  })

  useEffect(() => {
    let stream: MediaStream | null = null
    let timer = 0
    let stopped = false
    const recent = new Map<string, number>()

    const found = (text: string) => {
      const now = Date.now()
      if ((recent.get(text) ?? 0) > now - 3000) return
      recent.set(text, now)
      navigator.vibrate?.(80)
      handler.current(text)
    }

    async function start() {
      if (!navigator.mediaDevices?.getUserMedia) {
        setError('Este navegador não abre a câmera. Digite o código.')
        return
      }
      try {
        stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' }, audio: false })
      } catch {
        setError('Sem acesso à câmera. Libere a câmera nas configurações do navegador ou digite o código.')
        return
      }
      if (stopped || !video.current) {
        // Fechou enquanto a câmera ligava: a limpeza ainda não tinha o stream.
        stream.getTracks().forEach((t) => t.stop())
        return
      }
      video.current.srcObject = stream
      await video.current.play().catch(() => undefined)

      const Native = (window as unknown as { BarcodeDetector?: DetectorClass }).BarcodeDetector
      let read: (el: HTMLVideoElement) => Promise<string | null>
      try {
        read = Native ? nativeReader(new Native({ formats: ['qr_code'] })) : await jsQRReader()
      } catch {
        // O leitor não baixou (sem sinal): a câmera ligada não serviria para nada.
        stream.getTracks().forEach((t) => t.stop())
        setError('Não deu para ligar o leitor de QR-code. Digite o código.')
        return
      }
      const tick = async () => {
        if (stopped) return
        const el = video.current
        if (el && el.readyState >= 2) {
          const text = await read(el).catch(() => null)
          if (text) found(text)
        }
        timer = window.setTimeout(() => void tick(), 200)
      }
      void tick()
    }

    void start()
    return () => {
      stopped = true
      window.clearTimeout(timer)
      stream?.getTracks().forEach((t) => t.stop())
    }
  }, [])

  return (
    <div className="flex flex-col gap-3">
      <div className="relative overflow-hidden rounded-xl bg-slate-900">
        <video ref={video} muted playsInline className="aspect-square w-full object-cover" aria-label="Câmera" />
        <div className="pointer-events-none absolute inset-10 rounded-xl border-4 border-white/80" />
      </div>
      {error ? (
        <p role="alert" className="text-danger-fg">
          {error}
        </p>
      ) : (
        <p className="text-sm text-slate-600">Aponte para o QR-code da etiqueta. Pode bipar um pacote atrás do outro.</p>
      )}
      <button
        type="button"
        onClick={onClose}
        className="min-h-11 rounded-lg border border-slate-300 bg-white px-4 font-semibold text-brand-700"
      >
        Fechar câmera
      </button>
    </div>
  )
}

function nativeReader(detector: Detector) {
  return async (el: HTMLVideoElement) => (await detector.detect(el))[0]?.rawValue ?? null
}

// O jsQR só é baixado quando o navegador não tem leitor próprio.
async function jsQRReader() {
  const { default: jsQR } = await import('jsqr')
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  return async (el: HTMLVideoElement) => {
    if (!ctx) return null
    // Reduz o quadro: o QR da etiqueta é grande e a leitura fica mais rápida.
    const scale = Math.min(1, 640 / Math.max(el.videoWidth, el.videoHeight))
    canvas.width = Math.round(el.videoWidth * scale)
    canvas.height = Math.round(el.videoHeight * scale)
    ctx.drawImage(el, 0, 0, canvas.width, canvas.height)
    const img = ctx.getImageData(0, 0, canvas.width, canvas.height)
    return jsQR(img.data, img.width, img.height, { inversionAttempts: 'dontInvert' })?.data ?? null
  }
}
