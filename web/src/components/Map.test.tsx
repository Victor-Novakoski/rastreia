import { fireEvent, render, screen } from '@testing-library/react'
import { StrictMode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { PinMap, StopsMap } from './Map'

describe('PinMap', () => {
  it('mostra o pino salvo, também montando duas vezes como no StrictMode', () => {
    const { container } = render(
      <StrictMode>
        <PinMap value={{ latitude: -23.55, longitude: -46.63 }} onChange={() => {}} label="Mapa com o local da entrega" />
      </StrictMode>,
    )
    expect(container.querySelectorAll('.map-pin')).toHaveLength(1)
  })

  it('pelo teclado, Enter põe o pino no centro do mapa', () => {
    const onChange = vi.fn()
    render(<PinMap value={null} onChange={onChange} label="Mapa com o local da entrega" />)
    const map = screen.getByRole('application', { name: 'Mapa com o local da entrega' })
    expect(map).toHaveAccessibleDescription('Use as setas para mover o mapa e Enter para pôr o pino no centro.')
    fireEvent.keyDown(map, { key: 'Enter' })
    // Sem pino ainda, o mapa começa no centro do Brasil.
    const [[at]] = onChange.mock.calls
    expect(at.latitude).toBeCloseTo(-14.2)
    expect(at.longitude).toBeCloseTo(-51.9)
  })
})

describe('StopsMap', () => {
  it('parada feita leva um ✓, não só outra cor', () => {
    const { container } = render(
      <StopsMap
        label="Mapa com as paradas numeradas"
        stops={[
          { number: 1, latitude: -23.55, longitude: -46.63, done: true },
          { number: 2, latitude: -23.56, longitude: -46.64, done: false },
        ]}
      />,
    )
    expect(screen.getByRole('region', { name: 'Mapa com as paradas numeradas' })).toBeInTheDocument()
    expect([...container.querySelectorAll('.map-stop span')].map((s) => s.textContent)).toEqual(['✓', '2'])
  })
})
