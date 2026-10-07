import { useState } from 'react'
import { useAuth } from './auth'

/**
 * Botão Sair. Sem resposta da API a sessão continua valendo, então a tela
 * também continua e avisa: `failed` fica true até a próxima tentativa.
 */
export function useLogout() {
  const { logout } = useAuth()
  const [state, setState] = useState<'idle' | 'busy' | 'failed'>('idle')
  const exit = () => {
    setState('busy')
    logout().then(
      () => setState('idle'),
      () => setState('failed'),
    )
  }
  return { exit, busy: state === 'busy', failed: state === 'failed' }
}

export const logoutFailed = 'Não deu para sair agora. Confira a conexão e tente de novo.'
