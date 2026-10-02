import { useEffect, useRef, useState } from 'react'
import { connectLive, type LiveOptions } from './live'

/**
 * Mantém um WebSocket aberto enquanto o componente está na tela. `path` null
 * não conecta. Devolve se a conexão está aberta agora.
 */
export function useLive(path: string | null, opts: LiveOptions): boolean {
  const [connected, setConnected] = useState(false)
  // As callbacks mudam a cada render; a conexão não deve cair por isso.
  const optsRef = useRef(opts)
  useEffect(() => {
    optsRef.current = opts
  })

  const withToken = opts.token !== undefined
  useEffect(() => {
    if (!path) return
    const close = connectLive(path, {
      onMessage: (data) => optsRef.current.onMessage(data),
      onOpen: (again) => {
        setConnected(true)
        optsRef.current.onOpen?.(again)
      },
      onClose: () => {
        setConnected(false)
        optsRef.current.onClose?.()
      },
      token: withToken ? (renew) => optsRef.current.token?.(renew) ?? Promise.resolve(null) : undefined,
    })
    return () => {
      close()
      setConnected(false)
    }
  }, [path, withToken])

  return connected
}
