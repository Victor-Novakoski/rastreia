import { useQuery } from '@tanstack/react-query'
import { useAuth } from './auth'
import type { Role } from './session'

export type Me = {
  id: number
  name: string
  email: string
  role: Role
  carrier: { id: number; name: string; document: string | null }
}

/** Quem está logado e de qual transportadora, para o cabeçalho. */
export function useMe() {
  const { api } = useAuth()
  return useQuery({ queryKey: ['me'], queryFn: ({ signal }) => api<Me>('/me', { signal }), staleTime: 5 * 60_000 })
}
