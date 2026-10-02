import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Alert } from '../../components/Alert'
import { Button } from '../../components/Button'
import { TextField } from '../../components/Field'
import { Empty, LoadError, Loading } from '../../components/States'
import { ApiError, errorMessage } from '../../lib/api'
import { useAuth } from '../../lib/auth'
import { createDriver } from '../../lib/deliveries'
import { fieldErrors } from '../../lib/fields'
import { formatDateTime } from '../../lib/format'
import { useDrivers } from '../../lib/queries'

export function DriversPage() {
  const drivers = useDrivers()
  return (
    <>
      <h1 className="text-2xl font-bold">Motoristas</h1>
      <div className="mt-6 grid gap-6 lg:grid-cols-[1fr_22rem]">
        <section>
          {drivers.isPending ? (
            <Loading label="Carregando motoristas" />
          ) : drivers.isError ? (
            <LoadError error={drivers.error} onRetry={() => void drivers.refetch()} />
          ) : drivers.data.length === 0 ? (
            <Empty>Nenhum motorista cadastrado ainda.</Empty>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-3 py-2 font-semibold">Nome</th>
                    <th className="px-3 py-2 font-semibold">E-mail</th>
                    <th className="px-3 py-2 font-semibold">Desde</th>
                  </tr>
                </thead>
                <tbody>
                  {drivers.data.map((d) => (
                    <tr key={d.id} className="border-b border-slate-100 last:border-0">
                      <td className="px-3 py-2">{d.name}</td>
                      <td className="px-3 py-2">{d.email}</td>
                      <td className="px-3 py-2 whitespace-nowrap">{formatDateTime(d.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
        <div className="flex flex-col gap-4">
          <section className="rounded-lg border border-slate-200 bg-white p-4">
            <h2 className="text-lg font-semibold">Novo motorista</h2>
            <NewDriverForm />
          </section>
          <DriverAppHint />
        </div>
      </div>
    </>
  )
}

/** Onde o motorista entra: a transportadora passa esse endereço junto com a senha. */
function DriverAppHint() {
  const url = `${window.location.origin}/motorista/entrar`
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4 text-sm">
      <h2 className="font-semibold">Como o motorista entra</h2>
      <p className="mt-1 text-slate-600">Ele abre este endereço no celular e usa o e-mail e a senha que você cadastrou:</p>
      <p className="mt-2 rounded-md bg-slate-100 px-3 py-2 font-mono break-all">{url}</p>
    </section>
  )
}

function NewDriverForm() {
  const { api } = useAuth()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [created, setCreated] = useState('')
  const create = useMutation({
    mutationFn: () => createDriver(api, { name: name.trim(), email: email.trim(), password }),
    onMutate: () => setCreated(''),
    onSuccess: (d) => {
      void queryClient.invalidateQueries({ queryKey: ['drivers'] })
      setCreated(d.name)
      setName('')
      setEmail('')
      setPassword('')
    },
  })
  const errors = fieldErrors(create.error)
  const emailTaken = create.error instanceof ApiError && create.error.status === 409

  function submit(e: FormEvent) {
    e.preventDefault()
    if (!create.isPending) create.mutate()
  }

  return (
    <form onSubmit={submit} className="mt-3 flex flex-col gap-4">
      {created && <Alert tone="success">{created} foi cadastrado.</Alert>}
      {create.isError && !emailTaken && <Alert>{errorMessage(create.error)}</Alert>}
      <TextField label="Nome" value={name} onChange={(e) => setName(e.target.value)} required maxLength={120} error={errors.name} />
      <TextField
        label="E-mail"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        required
        autoComplete="off"
        error={emailTaken ? 'Já existe uma conta com esse e-mail.' : errors.email}
      />
      <TextField
        label="Senha inicial"
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
        minLength={10}
        autoComplete="new-password"
        error={errors.password}
        hint="Mínimo de 10 caracteres. Passe para o motorista por um canal seguro."
      />
      <Button type="submit" loading={create.isPending} className="self-start">
        Cadastrar
      </Button>
    </form>
  )
}
