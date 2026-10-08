import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { Button } from '../components/Button'
import { PublicLayout } from '../components/PublicLayout'
import { isValidCode, normalizeCode } from '../lib/tracking'

export function TrackingSearchPage() {
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')

  function submit(e: FormEvent) {
    e.preventDefault()
    const normalized = normalizeCode(code)
    if (!isValidCode(normalized)) {
      setError('O código tem 12 caracteres e começa com RS, como RS7K2M9QXA4P.')
      return
    }
    navigate(`/rastreio/${normalized}`)
  }

  return (
    <PublicLayout>
      <h1 className="text-2xl font-bold">Rastrear entrega</h1>
      <p className="mt-1 text-slate-600">Digite o código que você recebeu por e-mail.</p>
      <form onSubmit={submit} noValidate className="mt-6 flex flex-col gap-3">
        <label htmlFor="code" className="font-medium">
          Código de rastreio
        </label>
        <input
          id="code"
          name="code"
          value={code}
          onChange={(e) => {
            setCode(e.target.value)
            setError('')
          }}
          placeholder="RS7K2M9QXA4P"
          autoComplete="off"
          autoCapitalize="characters"
          spellCheck={false}
          maxLength={20}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'code-error' : undefined}
          className="min-h-11 rounded-lg border border-slate-300 bg-white px-3 font-mono text-lg uppercase tracking-wider focus:border-brand-600 focus:outline-2 focus:outline-brand-600 aria-invalid:border-danger-fg"
        />
        {error && (
          <p id="code-error" role="alert" className="text-sm text-danger-fg">
            {error}
          </p>
        )}
        <Button type="submit">Rastrear</Button>
      </form>
    </PublicLayout>
  )
}
