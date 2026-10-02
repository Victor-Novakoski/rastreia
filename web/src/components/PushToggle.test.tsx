import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as push from '../lib/push'
import { ApiError } from '../lib/api'
import { PushToggle } from './PushToggle'

afterEach(() => vi.restoreAllMocks())

describe('PushToggle', () => {
  it('some quando o navegador não tem push', async () => {
    const { container } = render(<PushToggle code="RS7K2M9QXA4P" />)
    await Promise.resolve()
    expect(container).toBeEmptyDOMElement()
  })

  it('explica como instalar no iPhone', async () => {
    vi.spyOn(push, 'pushState').mockResolvedValue('ios-install')
    render(<PushToggle code="RS7K2M9QXA4P" />)
    expect(await screen.findByText('Adicionar à Tela de Início')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('ativa e desativa', async () => {
    vi.spyOn(push, 'pushState').mockResolvedValue('off')
    const enable = vi.spyOn(push, 'enablePush').mockResolvedValue('on')
    vi.spyOn(push, 'disablePush').mockResolvedValue('off')
    render(<PushToggle code="RS7K2M9QXA4P" />)

    await userEvent.click(await screen.findByRole('button', { name: 'Ativar avisos' }))
    expect(enable).toHaveBeenCalledWith('RS7K2M9QXA4P')
    await userEvent.click(await screen.findByRole('button', { name: 'Desativar avisos' }))
    expect(await screen.findByRole('button', { name: 'Ativar avisos' })).toBeInTheDocument()
  })

  it('409 diz que a entrega não aceita mais avisos', async () => {
    vi.spyOn(push, 'pushState').mockResolvedValue('off')
    vi.spyOn(push, 'enablePush').mockRejectedValue(new ApiError(409, 'conflict'))
    render(<PushToggle code="RS7K2M9QXA4P" />)
    await userEvent.click(await screen.findByRole('button', { name: 'Ativar avisos' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Esta entrega não aceita mais avisos.')
  })
})
