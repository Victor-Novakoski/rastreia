import { Link } from 'react-router'
import { PublicLayout } from '../components/PublicLayout'

export function NotFoundPage() {
  return (
    <PublicLayout>
      <h1 className="text-2xl font-bold">Página não encontrada</h1>
      <Link to="/" className="mt-4 inline-block font-semibold text-brand-700 underline underline-offset-2">
        Voltar para o início
      </Link>
    </PublicLayout>
  )
}
