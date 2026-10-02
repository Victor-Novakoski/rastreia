import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { Alert } from '../../components/Alert'
import { DeliveryForm } from '../../components/DeliveryForm'
import { LoadError, Loading } from '../../components/States'
import { ApiError, errorMessage } from '../../lib/api'
import { useAuth } from '../../lib/auth'
import { createDelivery, type DeliveryInput } from '../../lib/deliveries'
import { fieldErrors } from '../../lib/fields'
import { useDrivers } from '../../lib/queries'

export function NewDeliveryPage() {
  const { api } = useAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const drivers = useDrivers()
  // Uma chave por formulário: reenviar depois de uma falha de rede não cria
  // outra entrega. Se a API respondeu com erro, nada foi criado e os dados vão
  // mudar, então a próxima tentativa usa uma chave nova.
  const [key, setKey] = useState(() => crypto.randomUUID())

  const create = useMutation({
    mutationFn: (input: DeliveryInput) => createDelivery(api, input, key),
    onSuccess: (d) => {
      void queryClient.invalidateQueries({ queryKey: ['deliveries'] })
      navigate(`/transportadora/entregas/${d.id}`, { state: { created: true } })
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status !== 0) setKey(crypto.randomUUID())
    },
  })
  const errors = fieldErrors(create.error)

  return (
    <div className="max-w-xl">
      <Link to="/transportadora/entregas" className="text-sm font-medium text-brand-700 hover:underline">
        ← Entregas
      </Link>
      <h1 className="mt-2 text-2xl font-bold">Nova entrega</h1>
      <div className="mt-6 flex flex-col gap-4">
        {create.isError && <Alert>{errorMessage(create.error)}</Alert>}
        {drivers.isPending ? (
          <Loading />
        ) : drivers.isError ? (
          <LoadError error={drivers.error} onRetry={() => void drivers.refetch()} />
        ) : (
          <DeliveryForm
            drivers={drivers.data}
            errors={errors}
            sending={create.isPending}
            submitLabel="Criar entrega"
            onSubmit={(input) => create.mutate(input)}
          />
        )}
      </div>
    </div>
  )
}
