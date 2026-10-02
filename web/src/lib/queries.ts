import { useQuery } from '@tanstack/react-query'
import { useAuth } from './auth'
import { listDrivers } from './deliveries'

/** Lista de motoristas, usada no filtro, nos formulários e para mostrar nomes. */
export function useDrivers() {
  const { api } = useAuth()
  return useQuery({ queryKey: ['drivers'], queryFn: ({ signal }) => listDrivers(api, signal) })
}
