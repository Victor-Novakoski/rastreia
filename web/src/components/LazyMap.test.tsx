import { render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { PinMap } from './LazyMap'

// O pedaço do mapa não baixa, como sem sinal ou depois de uma atualização do
// site com a aba aberta.
vi.mock('./Map', () => {
  throw new TypeError('Failed to fetch dynamically imported module')
})

it('mapa que não baixou avisa no lugar dele e não derruba o resto da tela', async () => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
  render(
    <>
      <PinMap value={null} onChange={() => {}} label="Mapa com o local da entrega" />
      <p>resto do formulário</p>
    </>,
  )
  expect(await screen.findByRole('alert')).toHaveTextContent('Não foi possível carregar o mapa.')
  expect(screen.getByRole('alert')).toHaveTextContent('Dá para salvar a entrega sem o pino e marcar depois.')
  expect(screen.getByRole('button', { name: 'Recarregar a página' })).toBeInTheDocument()
  expect(screen.getByText('resto do formulário')).toBeInTheDocument()
})
