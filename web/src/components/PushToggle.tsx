import { useEffect, useState } from 'react'
import { ApiError } from '../lib/api'
import { disablePush, enablePush, pushState, type PushState } from '../lib/push'

/** Liga e desliga o aviso no celular para uma entrega (Web Push). */
export function PushToggle({ code }: { code: string }) {
  const [state, setState] = useState<PushState | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    pushState(code).then(
      (s) => alive && setState(s),
      () => alive && setState('unsupported'),
    )
    return () => {
      alive = false
    }
  }, [code])

  if (state === null || state === 'unsupported') return null

  async function toggle() {
    setBusy(true)
    setError('')
    try {
      setState(state === 'on' ? await disablePush(code) : await enablePush(code))
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 409
          ? 'Esta entrega não aceita mais avisos.'
          : 'Não foi possível ativar os avisos. Tente de novo.',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mt-6 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="font-semibold">Avisos no celular</h2>
      {state === 'ios-install' && (
        <p className="mt-1 text-sm text-slate-600">
          No iPhone, toque em Compartilhar e depois em <strong>Adicionar à Tela de Início</strong>. Abra o Rastreia
          por lá para ativar os avisos.
        </p>
      )}
      {state === 'denied' && (
        <p className="mt-1 text-sm text-slate-600">
          As notificações deste site estão bloqueadas. Libere nas configurações do navegador para receber avisos.
        </p>
      )}
      {(state === 'on' || state === 'off') && (
        <>
          <p className="mt-1 text-sm text-slate-600">
            {state === 'on'
              ? 'Você vai receber um aviso a cada mudança desta entrega.'
              : 'Receba um aviso neste aparelho a cada mudança da entrega.'}
          </p>
          <button
            type="button"
            onClick={toggle}
            disabled={busy}
            aria-pressed={state === 'on'}
            className="mt-3 inline-flex min-h-11 items-center rounded-lg border border-brand-700 px-4 font-semibold text-brand-700 hover:bg-brand-50 disabled:opacity-60"
          >
            {state === 'on' ? 'Desativar avisos' : 'Ativar avisos'}
          </button>
        </>
      )}
      {error && (
        <p role="alert" className="mt-2 text-sm font-medium text-danger-fg">
          {error}
        </p>
      )}
    </div>
  )
}
