import { request } from './api'

/** Situação do aviso no celular para uma entrega neste navegador. */
export type PushState =
  | 'unsupported' // navegador sem Web Push
  | 'ios-install' // iPhone fora da tela de início: o Safari só libera push no app instalado
  | 'denied' // o usuário bloqueou as notificações do site
  | 'off'
  | 'on'

const storageKey = 'rastreia:push'

function isIOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
}

function isStandalone(): boolean {
  return window.matchMedia?.('(display-mode: standalone)').matches || (navigator as { standalone?: boolean }).standalone === true
}

export function isSupported(): boolean {
  return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window
}

/** Códigos que este navegador segue. Só uma conveniência para a tela: quem decide é a API. */
function followed(): string[] {
  try {
    return JSON.parse(localStorage.getItem(storageKey) ?? '[]') as string[]
  } catch {
    return []
  }
}

function setFollowed(codes: string[]) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(codes))
  } catch {
    // Sem armazenamento a tela só não lembra que o aviso está ligado.
  }
}

export async function pushState(code: string): Promise<PushState> {
  if (isIOS() && !isStandalone()) return 'ios-install'
  if (!isSupported()) return 'unsupported'
  if (Notification.permission === 'denied') return 'denied'
  const reg = await navigator.serviceWorker.getRegistration()
  const sub = await reg?.pushManager.getSubscription()
  return sub && followed().includes(code) ? 'on' : 'off'
}

/** A chave pública VAPID vem em base64url; o PushManager quer os bytes. */
export function keyBytes(base64url: string): Uint8Array<ArrayBuffer> {
  const pad = '='.repeat((4 - (base64url.length % 4)) % 4)
  const raw = atob((base64url + pad).replace(/-/g, '+').replace(/_/g, '/'))
  const out = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i)
  return out
}

/** Pede permissão, inscreve o navegador e registra a inscrição na entrega. */
export async function enablePush(code: string): Promise<PushState> {
  const permission = await Notification.requestPermission()
  if (permission !== 'granted') return permission === 'denied' ? 'denied' : 'off'
  const reg = await navigator.serviceWorker.register('/sw.js')
  await navigator.serviceWorker.ready
  let sub = await reg.pushManager.getSubscription()
  if (!sub) {
    const { public_key } = await request<{ public_key: string }>('/public/push/key')
    sub = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: keyBytes(public_key) })
  }
  await request(`/public/tracking/${encodeURIComponent(code)}/push`, { method: 'POST', body: sub.toJSON() })
  setFollowed([...new Set([...followed(), code])])
  return 'on'
}

/** Para de avisar desta entrega. A inscrição do navegador fica para as outras. */
export async function disablePush(code: string): Promise<PushState> {
  const reg = await navigator.serviceWorker.getRegistration()
  const sub = await reg?.pushManager.getSubscription()
  if (sub) {
    await request(`/public/tracking/${encodeURIComponent(code)}/push`, {
      method: 'DELETE',
      body: { endpoint: sub.endpoint },
    })
  }
  const rest = followed().filter((c) => c !== code)
  setFollowed(rest)
  if (sub && rest.length === 0) await sub.unsubscribe()
  return 'off'
}
