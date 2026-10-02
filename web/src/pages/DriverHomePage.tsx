import { useAuth } from '../lib/auth'

// Provisória: o app do motorista é o próximo item da etapa 3.
export function DriverHomePage() {
  const { logout } = useAuth()
  return (
    <main className="mx-auto max-w-xl px-4 py-10">
      <h1 className="text-2xl font-bold">App do motorista</h1>
      <p className="mt-2 text-slate-600">Em construção.</p>
      <button type="button" onClick={() => void logout()} className="mt-4 font-semibold text-brand-700 underline underline-offset-2">
        Sair
      </button>
    </main>
  )
}
