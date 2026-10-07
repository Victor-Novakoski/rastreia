import { baseURL } from './api'

/** Código com que a API fecha a conexão quando o token falta, é inválido ou venceu. */
export const UNAUTHORIZED = 4001

const MAX_DELAY = 30_000

export type LiveOptions = {
  /** Mensagem já convertida de JSON. */
  onMessage: (data: unknown) => void
  /** Conectou; `again` é true depois de uma queda, quando algo pode ter se perdido. */
  onOpen?: (again: boolean) => void
  onClose?: () => void
  /**
   * Para rotas que exigem login: devolve o access token que vai na primeira
   * mensagem. `renew` pede um token novo, depois de a API recusar o atual.
   */
  token?: (renew: boolean) => Promise<string | null>
}

/** Troca http(s) por ws(s) na URL da API. */
export function liveURL(path: string): string {
  return baseURL.replace(/^http/, 'ws') + path
}

/**
 * Abre um WebSocket com a API e reconecta sozinho quando cai, esperando cada
 * vez mais (até 30s) para não martelar um servidor fora do ar. Devolve a
 * função que fecha de vez.
 */
export function connectLive(path: string, opts: LiveOptions): () => void {
  let ws: WebSocket | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  let delay = 1000
  let opened = false
  let renew = false
  let stopped = false

  const schedule = () => {
    if (stopped) return
    timer = setTimeout(() => void open(), delay)
    delay = Math.min(delay * 2, MAX_DELAY)
  }

  const open = async () => {
    let token: string | null = null
    if (opts.token) {
      try {
        token = await opts.token(renew)
      } catch {
        // A renovação falhou (sem rede, 429, erro na API), mas a sessão pode
        // continuar valendo: tenta de novo mais tarde, ainda pedindo token novo.
        schedule()
        return
      }
      if (stopped) return
      // Sem sessão não há o que fazer; a tela de login assume.
      if (!token) return
    }
    renew = false
    const socket = new WebSocket(liveURL(path))
    ws = socket
    socket.onopen = () => {
      delay = 1000
      if (token) socket.send(JSON.stringify({ token }))
      opts.onOpen?.(opened)
      opened = true
    }
    socket.onmessage = (ev: MessageEvent) => {
      try {
        opts.onMessage(JSON.parse(String(ev.data)))
      } catch {
        // Mensagem que não é JSON não vem da nossa API; ignora.
      }
    }
    socket.onclose = (ev: CloseEvent) => {
      if (ws !== socket) return
      ws = null
      opts.onClose?.()
      if (ev.code === UNAUTHORIZED) renew = true
      schedule()
    }
  }

  void open()
  return () => {
    stopped = true
    clearTimeout(timer)
    ws?.close()
    ws = null
  }
}
