import { useState, type FormEvent } from 'react'
import { Navigate } from 'react-router'
import { Alert } from '../components/Alert'
import { AuthCard, AuthLink } from '../components/AuthCard'
import { Button } from '../components/Button'
import { TextField } from '../components/Field'
import { ApiError, errorMessage } from '../lib/api'
import { homeFor, useAuth } from '../lib/auth'
import { fieldErrors, type FieldErrors } from '../lib/fields'

/** Cadastro aberto da transportadora: cria a conta e já entra no painel. */
export function SignUpPage() {
  const { state, signUp } = useAuth()
  const [form, setForm] = useState({ carrier_name: '', document: '', name: '', email: '', password: '' })
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState<FieldErrors>({})
  const [created, setCreated] = useState(false)

  // A sessão nova já leva para o painel; quem acabou de se cadastrar ganha as boas-vindas.
  if (state.status === 'authenticated') {
    return <Navigate to={homeFor(state.session.role)} replace state={created ? { welcome: true } : undefined} />
  }

  const set = (field: keyof typeof form) => (e: { target: { value: string } }) =>
    setForm((f) => ({ ...f, [field]: e.target.value }))

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (sending) return
    setSending(true)
    setError('')
    setErrors({})
    try {
      setCreated(true)
      await signUp({
        carrier_name: form.carrier_name.trim(),
        document: form.document.trim() || undefined,
        name: form.name.trim(),
        email: form.email.trim(),
        password: form.password,
      })
    } catch (err) {
      setCreated(false)
      if (err instanceof ApiError && err.status === 409) {
        setErrors(
          err.message.includes('CNPJ')
            ? { document: 'Já existe uma transportadora com esse CNPJ.' }
            : { email: 'Já existe uma conta com esse e-mail.' },
        )
      } else {
        setErrors(fieldErrors(err))
        setError(errorMessage(err))
      }
      setSending(false)
    }
  }

  return (
    <AuthCard
      eyebrow="Transportadora"
      title="Cadastre sua transportadora"
      subtitle="Em um minuto você cadastra motoristas, cria entregas e manda o link de rastreio para seus clientes."
      footer={
        <p>
          Já tem conta? <AuthLink to="/transportadora/entrar">Entrar no painel</AuthLink>
        </p>
      }
    >
      <form onSubmit={submit} className="flex flex-col gap-4">
        {error && <Alert>{error}</Alert>}
        <fieldset className="flex flex-col gap-4">
          <legend className="mb-3 text-sm font-semibold text-slate-600">Empresa</legend>
          <TextField
            label="Nome da transportadora"
            autoComplete="organization"
            required
            maxLength={120}
            value={form.carrier_name}
            onChange={set('carrier_name')}
            error={errors.carrier_name}
          />
          <TextField
            label="CNPJ (opcional)"
            inputMode="text"
            maxLength={18}
            placeholder="00.000.000/0000-00"
            value={form.document}
            onChange={set('document')}
            error={errors.document}
            hint="Aceita o formato novo, com letras."
          />
        </fieldset>
        <fieldset className="flex flex-col gap-4 border-t border-slate-200 pt-4">
          <legend className="mb-3 pt-4 text-sm font-semibold text-slate-600">Responsável</legend>
          <TextField
            label="Seu nome"
            autoComplete="name"
            required
            maxLength={120}
            value={form.name}
            onChange={set('name')}
            error={errors.name}
          />
          <TextField
            label="E-mail"
            type="email"
            autoComplete="email"
            required
            value={form.email}
            onChange={set('email')}
            error={errors.email}
          />
          <TextField
            label="Senha"
            type="password"
            autoComplete="new-password"
            required
            minLength={10}
            value={form.password}
            onChange={set('password')}
            error={errors.password}
            hint="Mínimo de 10 caracteres."
          />
        </fieldset>
        <Button type="submit" loading={sending}>
          Criar conta
        </Button>
      </form>
    </AuthCard>
  )
}
