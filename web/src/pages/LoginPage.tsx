import { useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router'
import { Alert } from '../components/Alert'
import { AuthCard, AuthLink } from '../components/AuthCard'
import { Button } from '../components/Button'
import { TextField } from '../components/Field'
import { ApiError, errorMessage } from '../lib/api'
import { homeFor, useAuth } from '../lib/auth'
import type { Role } from '../lib/session'

const copy = {
  carrier: {
    eyebrow: 'Transportadora',
    title: 'Entrar no painel',
    subtitle: 'Acompanhe as entregas, os motoristas e o que precisa de atenção hoje.',
  },
  driver: {
    eyebrow: 'Motorista',
    title: 'Entrar no app',
    subtitle: 'Veja suas entregas do dia e atualize o status com um toque.',
  },
}

/** Login de cada público. A API é a mesma; quem entra vai para a área do próprio papel. */
export function LoginPage({ audience }: { audience: Role }) {
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

  const text = copy[audience]
  return (
    <AuthCard
      eyebrow={text.eyebrow}
      title={text.title}
      subtitle={text.subtitle}
      footer={
        audience === 'carrier' ? (
          <>
            <p>
              Ainda não usa o Rastreia? <AuthLink to="/transportadora/cadastro">Cadastre sua transportadora</AuthLink>
            </p>
            <p>
              É motorista? <AuthLink to="/motorista/entrar">Entre pelo app do motorista</AuthLink>
            </p>
          </>
        ) : (
          <>
            <p>Sua conta é criada pela transportadora. Esqueceu a senha? Peça uma nova para ela.</p>
            <p>
              É transportadora? <AuthLink to="/transportadora/entrar">Entre no painel</AuthLink>
            </p>
          </>
        )
      }
    >
      <form onSubmit={submit} className="flex flex-col gap-4">
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
    </AuthCard>
  )
}
