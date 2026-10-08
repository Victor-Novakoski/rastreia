import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
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
    expect(screen.getByRole('link', { name: /Sem motorista\s*2/ })).toHaveAttribute(
      'href',
      '/transportadora/entregas?status=pending&driver=none',
    )
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

  it('"Sem motorista" da visão geral abre só as pendentes sem motorista', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /summary': () => [200, summary],
      'GET /drivers': () => [200, drivers],
      'GET /me': () => [200, me],
      'GET /deliveries': () => [200, [{ ...delivery, status: 'pending', driver_id: null }]],
    })
    renderApp('/transportadora')
    await userEvent.click(await screen.findByRole('link', { name: /Sem motorista\s*2/ }))
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()
    const query = new URLSearchParams(calls.findLast((c) => c.path.startsWith('/deliveries?'))!.path.split('?')[1])
    expect(query.get('status')).toBe('pending')
    expect(query.get('driver')).toBe('none')
    expect(screen.getByLabelText('Motorista')).toHaveValue('none')

    // Escolher um motorista troca o filtro e volta para a primeira página.
    await userEvent.selectOptions(screen.getByLabelText('Motorista'), 'João')
    await waitFor(() => expect(calls.at(-1)!.path).toContain('driver=2'))
  })

  it('Próxima espera a página chegar e a página vazia depois da última avisa', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, drivers],
      'GET /deliveries': () => [200, Array.from({ length: 20 }, (_, i) => ({ ...delivery, id: i + 1, tracking_code: `RS7K2M9QXA${String(i).padStart(2, '0')}` }))],
    })
    const apiFetch = vi.mocked(globalThis.fetch).getMockImplementation()!
    let answer!: () => void
    vi.mocked(globalThis.fetch).mockImplementation((input, init) => {
      if (!String(input).includes('page=2')) return apiFetch(input, init)
      calls.push({ method: 'GET', path: '/deliveries?page=2', body: undefined, headers: new Headers() })
      return new Promise<Response>((resolve) => (answer = () => resolve(Response.json([]))))
    })
    renderApp('/transportadora/entregas')
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA00' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Próxima' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Próxima' })).toBeDisabled())
    expect(screen.getByRole('button', { name: 'Anterior' })).toBeDisabled()
    expect(screen.getByRole('table').parentElement).toHaveAttribute('aria-busy', 'true')
    await userEvent.click(screen.getByRole('button', { name: 'Próxima' }))
    expect(calls.filter((c) => c.path.includes('page=3'))).toHaveLength(0)

    await act(async () => answer())
    expect(await screen.findByText('Não há mais entregas. Volte para a página anterior.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Anterior' })).toBeEnabled()
  })

  it('Sair sem resposta da API mantém a sessão e avisa', async () => {
    let logoutWorks = false
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /deliveries': () => [200, [delivery]],
      'GET /drivers': () => [200, drivers],
      'POST /auth/logout': () => (logoutWorks ? [204] : [503, { error: 'unavailable' }]),
    })
    renderApp('/transportadora/entregas')
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Sair' }))
    expect(await screen.findByText('Não deu para sair agora. Confira a conexão e tente de novo.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()

    logoutWorks = true
    await userEvent.click(screen.getByRole('button', { name: 'Sair' }))
    expect(await screen.findByRole('heading', { name: 'Entrar no painel' })).toBeInTheDocument()
    expect(calls.filter((c) => c.path === '/auth/logout')).toHaveLength(2)
  })

  it('a sessão que acaba leva junto o cache: a próxima conta não vê nada da anterior', async () => {
    let session: object | null = carrierSession
    let deliveries = [delivery]
    mockApi({
      'POST /auth/refresh': () => (session ? [200, session] : [401, { error: 'invalid session' }]),
      'POST /auth/login': () => {
        session = { ...carrierSession, token: 'outra-conta' }
        return [200, session]
      },
      'GET /deliveries': ({ headers }) =>
        headers.get('Authorization') === `Bearer ${carrierSession.token}` && session === null
          ? [401, { error: 'invalid token' }]
          : [200, deliveries],
      'GET /drivers': () => [200, drivers],
      'GET /summary': () => [200, summary],
      'GET /me': () => [200, me],
    })
    renderApp('/transportadora/entregas', undefined, { staleTime: 30_000, refetchOnWindowFocus: false })
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()

    // A sessão venceu (ou saiu em outra aba): a próxima chamada recebe 401.
    session = null
    deliveries = []
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'failed')
    expect(await screen.findByRole('heading', { name: 'Entrar no painel' })).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('E-mail'), 'outra@example.com')
    await userEvent.type(screen.getByLabelText('Senha'), 'senha-da-outra')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    await userEvent.click(await screen.findByRole('link', { name: 'Entregas' }))
    expect(await screen.findByText('Nenhuma entrega cadastrada ainda.')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'RS7K2M9QXA4P' })).not.toBeInTheDocument()
  })

  it('busca por código, nome ou e-mail vai para a API e apagar mostra todas', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      // O handler só recebe o caminho; a query string está na última chamada.
      'GET /deliveries': () => [200, calls.at(-1)!.path.includes('q=') ? [] : [delivery]],
      'GET /drivers': () => [200, drivers],
    })
    renderApp('/transportadora/entregas?status=in_transit&page=2')
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()

    const box = screen.getByRole('searchbox', { name: 'Buscar' })
    await userEvent.type(box, '  joão {Enter}')
    expect(await screen.findByText('Nenhuma entrega encontrada para “joão”.')).toBeInTheDocument()
    const query = new URLSearchParams(calls.at(-1)!.path.split('?')[1])
    expect(query.get('q')).toBe('joão')
    expect(query.get('status')).toBe('in_transit')
    expect(query.get('page')).toBe('1')
    expect(box).toHaveFocus()

    await userEvent.clear(box)
    expect(await screen.findByRole('link', { name: 'RS7K2M9QXA4P' })).toBeInTheDocument()
    expect(calls.at(-1)!.path).not.toContain('q=')
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

  it('mudar o número manda o endereço inteiro, sem o pino antigo', async () => {
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
    expect(screen.getByRole('button', { name: 'Tirar o pino' })).toBeInTheDocument()
    await userEvent.clear(number)
    await userEvent.type(number, '20')
    // O pino era do número 10: a rota levaria o motorista ao lugar antigo.
    expect(screen.getByText('O endereço mudou, então o pino saiu do mapa. Ache de novo ou toque no mapa.')).toBeInTheDocument()
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
      latitude: null,
      longitude: null,
    })
  })

  it('só vale a resposta do último CEP digitado', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, carrierSession], 'GET /drivers': () => [200, drivers] })
    const api = vi.mocked(globalThis.fetch).getMockImplementation()!
    let answerFirst = () => {}
    const viaCEP = (body: object) => new Response(JSON.stringify(body), { headers: { 'Content-Type': 'application/json' } })
    vi.mocked(globalThis.fetch).mockImplementation(async (input, init) => {
      const url = String(input)
      if (url.includes('/ws/01001000/')) {
        // O CEP errado responde por último, numa conexão lenta.
        await new Promise<void>((resolve) => (answerFirst = resolve))
        return viaCEP({ logradouro: 'Praça da Sé', bairro: 'Sé', localidade: 'São Paulo', uf: 'SP' })
      }
      if (url.includes('/ws/01310100/')) {
        return viaCEP({ logradouro: 'Avenida Paulista', bairro: 'Bela Vista', localidade: 'São Paulo', uf: 'SP' })
      }
      return api(input, init)
    })
    renderApp('/transportadora/entregas/nova')
    const cep = await screen.findByLabelText('CEP')
    await userEvent.type(cep, '01001000')
    await userEvent.clear(cep)
    await userEvent.type(cep, '01310100')
    await waitFor(() => expect(screen.getByLabelText('Rua')).toHaveValue('Avenida Paulista'))
    await act(async () => answerFirst())
    expect(screen.getByLabelText('Rua')).toHaveValue('Avenida Paulista')
    expect(screen.getByLabelText('Bairro')).toHaveValue('Bela Vista')
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

describe('motoristas', () => {
  it('cadastra, limpa o formulário e mostra na lista; e-mail repetido aparece no campo', async () => {
    let list: typeof drivers = []
    const calls = mockApi({
      'POST /auth/refresh': () => [200, carrierSession],
      'GET /drivers': () => [200, list],
      'POST /drivers': ({ body }) => {
        const { name, email } = body as { name: string; email: string }
        if (list.some((d) => d.email === email)) return [409, { error: 'e-mail already in use' }]
        const created = { id: 2, name, email, role: 'driver', created_at: '2026-10-07T09:00:00Z' }
        list = [created]
        return [201, created]
      },
    })
    renderApp('/transportadora/motoristas')
    expect(await screen.findByText('Nenhum motorista cadastrado ainda.')).toBeInTheDocument()

    const fill = async () => {
      await userEvent.type(screen.getByLabelText('Nome'), 'João')
      await userEvent.type(screen.getByLabelText('E-mail'), 'joao@example.com')
      await userEvent.type(screen.getByLabelText('Senha inicial'), 'senha-do-joao')
      await userEvent.click(screen.getByRole('button', { name: 'Cadastrar' }))
    }
    await fill()
    expect(await screen.findByText('João foi cadastrado.')).toBeInTheDocument()
    expect(await screen.findByRole('cell', { name: 'joao@example.com' })).toBeInTheDocument()
    expect(screen.getByLabelText('Nome')).toHaveValue('')
    expect(screen.getByLabelText('Senha inicial')).toHaveValue('')
    expect(calls.find((c) => c.method === 'POST' && c.path === '/drivers')!.body).toEqual({
      name: 'João',
      email: 'joao@example.com',
      password: 'senha-do-joao',
    })

    await fill()
    expect(await screen.findByText('Já existe uma conta com esse e-mail.')).toBeInTheDocument()
    expect(screen.getByLabelText('E-mail')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.queryByText('João foi cadastrado.')).not.toBeInTheDocument()
  })
})
