/**
 * WebSocket de mentira para os testes: nunca abre conexão de verdade. O teste
 * pega a instância em `FakeWebSocket.instances` e faz o papel do servidor.
 */
export class FakeWebSocket {
  static instances: FakeWebSocket[] = []

  readonly url: string
  sent: string[] = []
  closed = false
  onopen: (() => void) | null = null
  onmessage: ((ev: { data: string }) => void) | null = null
  onclose: ((ev: { code: number }) => void) | null = null

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.closed = true
  }

  // Lado do servidor.
  open() {
    this.onopen?.()
  }

  receive(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) })
  }

  drop(code = 1006) {
    this.closed = true
    this.onclose?.({ code })
  }

  /** A última conexão aberta para um caminho da API. */
  static last(path: string): FakeWebSocket | undefined {
    return FakeWebSocket.instances.findLast((ws) => new URL(ws.url).pathname === path)
  }
}
