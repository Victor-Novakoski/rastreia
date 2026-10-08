/** Link público de rastreio, que vai no QR-code da etiqueta. */
export function trackingURL(code: string): string {
  return `${window.location.origin}/rastreio/${code}`
}

/**
 * QR-code como imagem SVG em data URL: entra num <img> sem HTML injetado e a
 * CSP já aceita `data:` em imagens.
 */
export async function qrDataURL(text: string): Promise<string> {
  const { default: QRCode } = await import('qrcode')
  const svg = await QRCode.toString(text, { type: 'svg', errorCorrectionLevel: 'M', margin: 1 })
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}
