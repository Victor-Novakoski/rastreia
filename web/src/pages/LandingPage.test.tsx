import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { mockApi, renderApp } from '../test/render'

describe('página inicial', () => {
  it('leva cada público para a sua área', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, {}] })
    renderApp('/')
    expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('Seu cliente sabe onde está a encomenda')
    expect(screen.getByRole('link', { name: 'Criar conta' })).toHaveAttribute('href', '/transportadora/cadastro')
    expect(screen.getByRole('link', { name: 'Entrar no app' })).toHaveAttribute('href', '/motorista/entrar')
    expect(screen.getAllByRole('link', { name: 'Entrar' }).map((a) => a.getAttribute('href'))).toContain('/transportadora/entrar')
  })

  it('rastreia direto pelo código', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, {}], 'GET /public/tracking/RS7K2M9QXA4P': () => [404, {}] })
    renderApp('/')
    await userEvent.type(await screen.findByLabelText('Código de rastreio'), 'rs7k-2m9q-xa4p')
    await userEvent.click(screen.getByRole('button', { name: 'Rastrear' }))
    expect(await screen.findByText('Buscar outro código')).toBeInTheDocument()
  })

  it('quem já está logado vai para a própria área', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, { token: 't', role: 'driver', expires_in: 900 }],
      'GET /me/deliveries': () => [200, []],
    })
    renderApp('/')
    expect(await screen.findByRole('heading', { name: 'Para fazer (0)' })).toBeInTheDocument()
  })

  it('o motorista tem a própria tela de login', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, {}] })
    renderApp('/motorista')
    expect(await screen.findByRole('heading', { name: 'Entrar no app' })).toBeInTheDocument()
    expect(screen.getByText(/Sua conta é criada pela transportadora/)).toBeInTheDocument()
  })
})
