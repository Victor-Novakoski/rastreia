import { useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router'
import { Alert } from '../components/Alert'
import { Button } from '../components/Button'
import { TextField } from '../components/Field'
import { ApiError, errorMessage } from '../lib/api'
import { homeFor, useAuth } from '../lib/auth'

export function LoginPage() {
  const { state, login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')

  if (state.status === 'authenticated') return <Navigate to={homeFor(state.session.role)} replace />

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (sending) return
    setSending(true)
    setError('')
    try {
      const role = await login(email, password)
      navigate(homeFor(role), { replace: true })
    } catch (err) {
      setError(err instanceof ApiError && err.status === 401 ? 'E-mail ou senha incorretos.' : errorMessage(err))
      setSending(false)
    }
  }

  return (
    <div className="flex min-h-dvh items-center justify-center px-4">
      <main className="w-full max-w-sm rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-lg font-bold text-brand-800">Rastreia</p>
        <h1 className="mt-1 text-2xl font-bold">Entrar</h1>
        <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
          {error && <Alert>{error}</Alert>}
          <TextField
            label="E-mail"
            type="email"
            autoComplete="username"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <TextField
            label="Senha"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Button type="submit" loading={sending}>
            Entrar
          </Button>
        </form>
      </main>
    </div>
  )
}
