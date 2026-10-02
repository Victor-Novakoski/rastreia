import { useQuery } from '@tanstack/react-query'
import { useAuth } from './auth'
import { listMyDeliveries } from './deliveries'
import type { Status } from './status'

export function useMyDeliveries() {
  const { api } = useAuth()
  return useQuery({
    queryKey: ['me', 'deliveries'],
    queryFn: ({ signal }) => listMyDeliveries(api, signal),
    // Na rua o sinal cai: tenta de novo algumas vezes antes de mostrar o erro.
    retry: 3,
  })
}

/** O texto do botão diz o que o motorista fez, não o nome do status. */
export const actionLabel: Record<Status, string> = {
  pending: 'Aguardando coleta',
  picked_up: 'Confirmar coleta',
  in_transit: 'Saí para entrega',
  delivered: 'Entreguei',
  failed: 'Não consegui entregar',
}

export const retryLabel = 'Nova tentativa'
