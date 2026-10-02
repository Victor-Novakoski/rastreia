import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { carrierSession, mockApi, renderApp } from '../../test/render'
import { FakeWebSocket } from '../../test/websocket'

const delivery = {
  id: 1,
  tracking_code: 'RS7K2M9QXA4P',
  recipient_name: 'Maria Souza',
  recipient_email: 'maria@example.com',
  address: 'Rua A, 10',
  // Entrega antiga: só o endereço numa linha.
  recipient_phone: '',
  postal_code: '',
  street: '',
  number: '',
  complement: '',
  district: '',
  city: '',
  state: '',
  address_reference: '',
  latitude: null,
  longitude: null,
  status: 'in_transit',
  driver_id: 2,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T15:00:00Z',
  completed_at: null,
  anonymized_at: null,
}
const drivers = [{ id: 2, name: 'João', email: 'joao@example.com', role: 'driver', created_at: '2026-10-01T09:00:00Z' }]

const me = {
  id: 1,
  name: 'Carla Dias',
  email: 'carla@example.com',
  role: 'carrier',
  carrier_id: 1,
  created_at: '2026-10-01T09:00:00Z',
  carrier: { id: 1, name: 'Expresso Sul', document: null },
}
const summary = {
  since: '2026-09-02T00:00:00Z',
  by_status: { pending: 3, picked_up: 1, in_transit: 2, delivered: 10, failed: 1 },
  unassigned: 2,
}

describe('login', () => {
  it('sem sessão, o painel manda para o login da transportadora', async () => {
    mockApi({ 'POST /auth/refresh': () => [401, { error: 'invalid session' }] })
    renderApp('/transportadora/entregas')
    expect(await screen.findByRole('heading', { name: 'Entrar no painel' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Cadastre sua transportadora' })).toHaveAttribute('href', '/transportadora/cadastro')
  })

  it('senha errada mostra a mensagem e libera o botão', async () => {
    mockApi({
      'POST /auth/refresh': () => [401, {}],
      'POST /auth/login': () => [401, { error: 'invalid e-mail or password' }],
    })
    renderApp('/transportadora/entrar')
    await userEvent.type(await screen.findByLabelText('E-mail'), 'carla@example.com')
    await userEvent.type(screen.getByLabelText('Senha'), 'errada')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByText('E-mail ou senha incorretos.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Entrar' })).toBeEnabled()
  })

  it('transportadora entra e vê a visão geral, com o token só no cabeçalho', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [401, {}],
      'POST /auth/login': () => [200, carrierSession],
      'GET /summary': () => [200, summary],
      'GET /drivers': () => [200, drivers],
      'GET /me': () => [200, me],
    })
    renderApp('/transportadora/entrar')
    await userEvent.type(await screen.findByLabelText('E-mail'), 'carla@example.com')
    await userEvent.type(screen.getByLabelText('Senha'), 'senha-da-carla')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))

    expect(await screen.findByRole('heading', { name: 'Olá, Carla' })).toBeInTheDocument()
    expect(await screen.findByText('Expresso Sul')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Em andamento\s*6/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Sem motorista\s*2/ })).toHaveAttribute('href', '/transportadora/entregas?status=pending')
    expect(screen.queryByRole('heading', { name: 'Primeiros passos' })).not.toBeInTheDocument()
    const call = calls.find((c) => c.path === '/summary')!
    expect(call.headers.get('Authorization')).toBe('Bearer access-token')
    expect(call.credentials).toBe('omit')
    expect(calls.find((c) => c.path === '/auth/login')!.credentials).toBe('include')
  })
})

describe('cadastro', () => {
  it('cria a transportadora, entra logado e mostra os primeiros passos', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [401, {}],
      'POST /auth/signup': () => [200, carrierSession],
      'GET /summary': () => [200, { ...summary, by_status: { pending: 0, picked_up: 0, in_transit: 0, delivered: 0, failed: 0 }, unassigned: 0 }],
      'GET /drivers': () => [200, []],
      'GET /me': () => [200, me],
    })
    renderApp('/transportadora/cadastro')
    await userEvent.type(await screen.findByLabelText('Nome da transportadora'), ' Expresso Sul ')
    await userEvent.type(screen.getByLabelText('Seu nome'), 'Carla Dias')
    await userEvent.type(screen.getByLabelText('E-mail'), 'carla@example.com')
    await userEvent.type(screen.getByLabelText('Senha'), 'transporte-forte')
    await userEvent.click(screen.getByRole('button', { name: 'Criar conta' }))

    expect(await screen.findByRole('heading', { name: 'Primeiros passos' })).toBeInTheDocument()
    expect(screen.getByText(/Conta criada/)).toBeInTheDocument()
    const post = calls.find((c) => c.path === '/auth/signup')!
    expect(post.credentials).toBe('include')
    expect(post.body).toEqual({
      carrier_name: 'Expresso Sul',
      name: 'Carla Dias',
      email: 'carla@example.com',
      password: 'transporte-forte',
    })
  })

  it('CNPJ repetido aparece no campo', async () => {
    mockApi({
      'POST /auth/refresh': () => [401, {}],
      'POST /auth/signup': () => [409, { error: 'conflict: CNPJ already in use' }],
    })
    renderApp('/transportadora/cadastro')
    await userEvent.type(await screen.findByLabelText('Nome da transportadora'), 'Expresso Sul')
    await userEvent.type(screen.getByLabelText('CNPJ (opcional)'), '11.222.333/0001-81')
    await userEvent.type(screen.getByLabelText('Seu nome'), 'Carla')
    await userEvent.type(screen.getByLabelText('E-mail'), 'carla@example.com')
    await userEvent.type(screen.getByLabelText('Senha'), 'transporte-forte')
    await userEvent.click(screen.getByRole('button', { name: 'Criar conta' }))

    expect(await screen.findByText('Já existe uma transportadora com esse CNPJ.')).toBeInTheDocument()
    expect(screen.getByLabelText('CNPJ (opcional)')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('button', { name: 'Criar conta' })).toBeEnabled()
  })
})

describe('painel', () => {
  it('token vencido é renovado uma vez e a chamada é repetida', async () => {
    let refreshes = 0
    const calls = mockApi({
      'POST /auth/refresh': () => {
        refreshes++
        return [200, { ...carrierSession, token: `token-${refreshes}` }]
      },
      'GET /deliveries': ({ headers }) =>
        headers.get('Authorization') === 'Bearer token-1' ? [401, { error: 'invalid token' }] : [200, [delivery]],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/transportadora/entregas')
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()
    const lists = calls.filter((c) => c.path.startsWith('/deliveries'))
    expect(lists.at(-1)!.headers.get('Authorization')).toBe('Bearer token-2')
  })

  it('filtro de status vai para a API', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /deliveries': () => [200, []],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/transportadora/entregas')
    expect(await screen.findByText('Nenhuma entrega cadastrada ainda.')).toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'failed')
    await waitFor(() => expect(calls.some((c) => c.path.includes('status=failed'))).toBe(true))
    expect(await screen.findByText('Nenhuma entrega com esse filtro.')).toBeInTheDocument()
  })

  it('nova entrega preenche pelo CEP, acha o ponto no mapa e manda Idempotency-Key', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, drivers],
      'GET /ws/01001000/json/': () => [
        200,
        { cep: '01001-000', logradouro: 'Praça da Sé', bairro: 'Sé', localidade: 'São Paulo', uf: 'SP' },
      ],
      'GET /search': () => [200, [{ lat: '-23.5503', lon: '-46.6339' }]],
      'POST /deliveries': () => [422, { error: 'invalid input', fields: { recipient_email: 'must be a valid e-mail' } }],
    })
    renderApp('/transportadora/entregas/nova')
    await userEvent.type(await screen.findByLabelText('Nome'), 'Maria')
    await userEvent.type(screen.getByLabelText('Telefone'), '11987654321')
    expect(screen.getByLabelText('Telefone')).toHaveValue('(11) 98765-4321')
    await userEvent.type(screen.getByLabelText('E-mail'), 'maria@x')
    await userEvent.type(screen.getByLabelText('CEP'), '01001000')
    await waitFor(() => expect(screen.getByLabelText('Rua')).toHaveValue('Praça da Sé'))
    expect(screen.getByLabelText('Bairro')).toHaveValue('Sé')
    expect(screen.getByLabelText('Cidade')).toHaveValue('São Paulo')
    expect(screen.getByLabelText('UF')).toHaveValue('SP')
    await waitFor(() => expect(screen.getByLabelText('Número')).toHaveFocus())
    await userEvent.type(screen.getByLabelText('Número'), '10')
    await userEvent.type(screen.getByLabelText('Ponto de referência'), 'Portão azul')
    await userEvent.click(screen.getByRole('button', { name: 'Achar no mapa' }))
    expect(await screen.findByRole('button', { name: 'Tirar o pino' })).toBeInTheDocument()
    const search = calls.find((c) => c.path.startsWith('/search'))!
    expect(new URLSearchParams(search.path.split('?')[1]).get('street')).toBe('10 Praça da Sé')

    await userEvent.click(screen.getByRole('button', { name: 'Criar entrega' }))
    expect(await screen.findByText('E-mail inválido.')).toBeInTheDocument()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/deliveries')!
    expect(post.headers.get('Idempotency-Key')).toMatch(/^[0-9a-f-]{36}$/)
    expect(post.body).toEqual({
      recipient_name: 'Maria',
      recipient_email: 'maria@x',
      recipient_phone: '11987654321',
      postal_code: '01001000',
      street: 'Praça da Sé',
      number: '10',
      complement: '',
      district: 'Sé',
      city: 'São Paulo',
      state: 'SP',
      address_reference: 'Portão azul',
      latitude: -23.5503,
      longitude: -46.6339,
    })
  })

  it('CEP inexistente pede o endereço à mão', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, drivers],
      'GET /ws/99999999/json/': () => [200, { erro: 'true' }],
    })
    renderApp('/transportadora/entregas/nova')
    await userEvent.type(await screen.findByLabelText('CEP'), '99999999')
    expect(await screen.findByText('CEP não encontrado. Preencha o endereço à mão.')).toBeInTheDocument()
  })

  it('entrega antiga troca de motorista sem preencher o endereço', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, [...drivers, { ...drivers[0], id: 3, name: 'Rita' }]],
      'GET /deliveries/1': () => [200, delivery],
      'GET /deliveries/1/events': () => [200, []],
      'PATCH /deliveries/1': () => [200, { ...delivery, driver_id: 3 }],
    })
    renderApp('/transportadora/entregas/1')
    expect(await screen.findByText(/Endereço atual: Rua A, 10/)).toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Motorista'), 'Rita')
    await userEvent.click(screen.getByRole('button', { name: 'Salvar' }))
    expect(await screen.findByText('Alterações salvas.')).toBeInTheDocument()
    expect(calls.find((c) => c.method === 'PATCH')!.body).toEqual({ driver_id: 3 })
  })

  it('mudar o número manda o endereço inteiro', async () => {
    const full = {
      ...delivery,
      recipient_phone: '11987654321',
      postal_code: '01001000',
      street: 'Praça da Sé',
      number: '10',
      district: 'Sé',
      city: 'São Paulo',
      state: 'SP',
      latitude: -23.55,
      longitude: -46.63,
    }
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, drivers],
      'GET /deliveries/1': () => [200, full],
      'GET /deliveries/1/events': () => [200, []],
      'PATCH /deliveries/1': () => [200, full],
    })
    renderApp('/transportadora/entregas/1')
    const number = await screen.findByLabelText('Número')
    await userEvent.clear(number)
    await userEvent.type(number, '20')
    await userEvent.click(screen.getByRole('button', { name: 'Salvar' }))
    await screen.findByText('Alterações salvas.')
    expect(calls.find((c) => c.method === 'PATCH')!.body).toEqual({
      postal_code: '01001000',
      street: 'Praça da Sé',
      number: '20',
      complement: '',
      district: 'Sé',
      city: 'São Paulo',
      state: 'SP',
      address_reference: '',
      latitude: -23.55,
      longitude: -46.63,
    })
  })

  it('etiqueta mostra o QR-code e o código', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /me': () => [200, me],
      'GET /deliveries/1': () => [200, { ...delivery, recipient_phone: '11987654321' }],
    })
    renderApp('/transportadora/entregas/1/etiqueta')
    const qr = await screen.findByRole('img', { name: 'QR-code da entrega RS7K2M9QXA4P' })
    expect(qr.getAttribute('src')).toMatch(/^data:image\/svg\+xml/)
    expect(screen.getByText('(11) 98765-4321')).toBeInTheDocument()
    expect(await screen.findByText('Expresso Sul')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Imprimir' })).toBeInTheDocument()
  })

  it('detalhe oferece só as transições válidas', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, drivers],
      'GET /deliveries/1': () => [200, delivery],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/transportadora/entregas/1')
    const select = await screen.findByLabelText('Novo status')
    const options = Array.from(select.querySelectorAll('option:not([disabled])')).map((o) => o.textContent)
    expect(options).toEqual(['Entregue', 'Não entregue'])
  })

  it('motorista não entra no painel', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, { ...carrierSession, role: 'driver' }], 'GET /me/deliveries': () => [200, []] })
    renderApp('/transportadora/entregas')
    expect(await screen.findByRole('heading', { name: 'Para fazer (0)' })).toBeInTheDocument()
  })

  it('recarrega a lista quando o WebSocket avisa de uma mudança', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /deliveries': () => [200, [delivery]],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/transportadora/entregas')
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()

    await waitFor(() => expect(FakeWebSocket.last('/live/deliveries')).toBeDefined())
    const ws = FakeWebSocket.last('/live/deliveries')!
    act(() => ws.open())
    expect(ws.sent).toEqual([JSON.stringify({ token: carrierSession.token })])
    expect(screen.getByRole('status')).toHaveTextContent('Ao vivo')

    const lists = () => calls.filter((c) => c.path.startsWith('/deliveries')).length
    const before = lists()
    act(() => ws.receive({ delivery_id: 1, status: 'delivered' }))
    await waitFor(() => expect(lists()).toBe(before + 1))
  })
})
