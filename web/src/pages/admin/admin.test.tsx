import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { adminSession, mockApi, renderApp } from '../../test/render'

const delivery = {
  id: 1,
  tracking_code: 'RS7K2M9QXA4P',
  recipient_name: 'Maria Souza',
  recipient_email: 'maria@example.com',
  address: 'Rua A, 10',
  status: 'in_transit',
  driver_id: 2,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T15:00:00Z',
  completed_at: null,
}
const drivers = [{ id: 2, name: 'João', email: 'joao@example.com', role: 'driver', created_at: '2026-10-01T09:00:00Z' }]

describe('login', () => {
  it('sem sessão, o painel manda para o login', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, { error: 'invalid session' }] })
    renderApp('/admin/entregas')
    expect(await screen.findByRole('heading', { name: 'Entrar' })).toBeInTheDocument()
  })

  it('senha errada mostra a mensagem e libera o botão', async () => {
    mockApi({
      'POST /auth/refresh': () => [401, {}],
      'POST /auth/login': () => [401, { error: 'invalid e-mail or password' }],
    })
    renderApp('/entrar')
    await userEvent.type(await screen.findByLabelText('E-mail'), 'admin@rastreia.dev')
    await userEvent.type(screen.getByLabelText('Senha'), 'errada')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByText('E-mail ou senha incorretos.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Entrar' })).toBeEnabled()
  })

  it('admin entra e vê as entregas, com o token só no cabeçalho', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [401, {}],
      'POST /auth/login': () => [200, adminSession],
      'GET /deliveries': () => [200, [delivery]],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/entrar')
    await userEvent.type(await screen.findByLabelText('E-mail'), 'admin@rastreia.dev')
    await userEvent.type(screen.getByLabelText('Senha'), 'admin12345')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))

    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()
    expect(await screen.findByText('João')).toBeInTheDocument()
    const list = calls.find((c) => c.path.startsWith('/deliveries'))!
    expect(list.headers.get('Authorization')).toBe('Bearer access-token')
    expect(list.credentials).toBe('omit')
    expect(calls.find((c) => c.path === '/auth/login')!.credentials).toBe('include')
  })
})

describe('painel', () => {
  it('token vencido é renovado uma vez e a chamada é repetida', async () => {
    let refreshes = 0
    const calls = mockApi({
      'POST /auth/refresh': () => {
        refreshes++
        return [200, { ...adminSession, token: `token-${refreshes}` }]
      },
      'GET /deliveries': ({ headers }) =>
        headers.get('Authorization') === 'Bearer token-1' ? [401, { error: 'invalid token' }] : [200, [delivery]],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/admin/entregas')
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()
    const lists = calls.filter((c) => c.path.startsWith('/deliveries'))
    expect(lists.at(-1)!.headers.get('Authorization')).toBe('Bearer token-2')
  })

  it('filtro de status vai para a API', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, adminSession],
      'GET /deliveries': () => [200, []],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/admin/entregas')
    expect(await screen.findByText('Nenhuma entrega cadastrada ainda.')).toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'failed')
    await waitFor(() => expect(calls.some((c) => c.path.includes('status=failed'))).toBe(true))
    expect(await screen.findByText('Nenhuma entrega com esse filtro.')).toBeInTheDocument()
  })

  it('nova entrega manda Idempotency-Key e mostra os erros de campo', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, adminSession],
      'GET /drivers': () => [200, drivers],
      'POST /deliveries': () => [422, { error: 'invalid input', fields: { recipient_email: 'must be a valid e-mail' } }],
    })
    renderApp('/admin/entregas/nova')
    await userEvent.type(await screen.findByLabelText('Nome do destinatário'), 'Maria')
    await userEvent.type(screen.getByLabelText('E-mail do destinatário'), 'maria@x')
    await userEvent.type(screen.getByLabelText('Endereço'), 'Rua A, 10')
    await userEvent.click(screen.getByRole('button', { name: 'Criar entrega' }))

    expect(await screen.findByText('E-mail inválido.')).toBeInTheDocument()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/deliveries')!
    expect(post.headers.get('Idempotency-Key')).toMatch(/^[0-9a-f-]{36}$/)
    expect(post.body).toEqual({ recipient_name: 'Maria', recipient_email: 'maria@x', address: 'Rua A, 10' })
  })

  it('detalhe oferece só as transições válidas', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, adminSession],
      'GET /drivers': () => [200, drivers],
      'GET /deliveries/1': () => [200, delivery],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/admin/entregas/1')
    const select = await screen.findByLabelText('Novo status')
    const options = Array.from(select.querySelectorAll('option:not([disabled])')).map((o) => o.textContent)
    expect(options).toEqual(['Entregue', 'Não entregue'])
  })

  it('motorista não entra no painel', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, { ...adminSession, role: 'driver' }], 'GET /me/deliveries': () => [200, []] })
    renderApp('/admin/entregas')
    expect(await screen.findByRole('heading', { name: 'Para fazer (0)' })).toBeInTheDocument()
  })
})
