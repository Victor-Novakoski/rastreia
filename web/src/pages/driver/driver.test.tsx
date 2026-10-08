import { fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { mockApi, renderApp } from '../../test/render'

const session = { token: 't', role: 'driver', expires_in: 900 }
const base = {
  recipient_email: 'x@example.com',
  driver_id: 2,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T12:00:00Z',
  completed_at: null,
  anonymized_at: null,
}
const pickedUp = { ...base, id: 1, tracking_code: 'RS7K2M9QXA4P', recipient_name: 'Maria', address: 'Rua A, 10', status: 'picked_up' }
const delivered = { ...base, id: 2, tracking_code: 'RSQ8W3ZK5MNB', recipient_name: 'Carlos', address: 'Rua B, 20', status: 'delivered' }

describe('app do motorista', () => {
  it('lista pendentes primeiro e guarda as entregues', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, session], 'GET /me/deliveries': () => [200, [delivered, pickedUp]] })
    renderApp('/motorista')
    expect(await screen.findByRole('heading', { name: 'Para fazer (1)' })).toBeInTheDocument()
    expect(screen.getByText('Entregues (1)')).toBeInTheDocument()
  })

  it('um toque muda o status', async () => {
    let current = pickedUp
    const calls = mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /deliveries/1': () => [200, current],
      'GET /deliveries/1/events': () => [200, []],
      'POST /deliveries/1/events': ({ body }) => {
        current = { ...current, status: (body as { status: string }).status }
        return [201, { id: 9, delivery_id: 1, status: current.status, note: null, created_by: 2, created_at: '2026-10-01T13:00:00Z' }]
      },
    })
    renderApp('/motorista/entregas/1')
    await userEvent.click(await screen.findByRole('button', { name: 'Saí para entrega' }))
    expect(await screen.findByText('Status atualizado: Em rota.')).toBeInTheDocument()
    expect(calls.find((c) => c.method === 'POST' && c.path === '/deliveries/1/events')!.body).toEqual({ status: 'in_transit' })
    expect(await screen.findByRole('button', { name: 'Entreguei' })).toBeInTheDocument()
  })

  it('falha pede o motivo', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /deliveries/1': () => [200, pickedUp],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/motorista/entregas/1')
    await userEvent.click(await screen.findByRole('button', { name: 'Não consegui entregar' }))
    expect(screen.getByLabelText('O que aconteceu?')).toBeRequired()
  })

  it('409 de uma repetição que já tinha chegado conta como sucesso', async () => {
    let reads = 0
    mockApi({
      'POST /auth/refresh': () => [200, session],
      // A segunda leitura já mostra o status novo.
      'GET /deliveries/1': () => [200, reads++ === 0 ? pickedUp : { ...pickedUp, status: 'in_transit' }],
      'GET /deliveries/1/events': () => [200, []],
      'POST /deliveries/1/events': () => [409, { error: 'status changed' }],
    })
    renderApp('/motorista/entregas/1')
    await userEvent.click(await screen.findByRole('button', { name: 'Saí para entrega' }))
    expect(await screen.findByText('Status atualizado: Em rota.')).toBeInTheDocument()
  })

  it('sem sinal mostra o aviso e não perde a tela', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /deliveries/1': () => [200, pickedUp],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/motorista/entregas/1')
    const button = await screen.findByRole('button', { name: 'Saí para entrega' })
    const { vi } = await import('vitest')
    vi.mocked(globalThis.fetch).mockRejectedValueOnce(new TypeError('Failed to fetch'))
    await userEvent.click(button)
    expect(await screen.findByText('Sem sinal. Quando voltar, toque de novo.')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Saí para entrega' })).toBeEnabled())
  })

  it('abre uma entrega que não está entre as 100 da lista', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/deliveries': () => [200, []],
      'GET /deliveries/1': () => [200, pickedUp],
      'GET /deliveries/1/events': () => [200, []],
    })
    renderApp('/motorista/entregas/1')
    expect(await screen.findByRole('heading', { name: 'Maria' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Saí para entrega' })).toBeInTheDocument()
  })

  it('entrega de outro motorista não é encontrada', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, session], 'GET /deliveries/9': () => [404, { error: 'not found' }] })
    renderApp('/motorista/entregas/9')
    expect(await screen.findByText('Entrega não encontrada.')).toBeInTheDocument()
  })
})

const parts = {
  recipient_phone: '11987654321',
  postal_code: '01001000',
  street: 'Praça da Sé',
  complement: '',
  district: 'Sé',
  city: 'São Paulo',
  state: 'SP',
  address_reference: '',
  latitude: -23.55,
  longitude: -46.63,
}
const pkg = (id: number, position: number, name: string, number: string, extra = {}) => ({
  ...base,
  ...parts,
  id,
  position,
  number,
  tracking_code: `RSABCDEFGHJ${id}`,
  recipient_name: name,
  address: `Praça da Sé, ${number} - Sé, São Paulo - SP, 01001-000`,
  status: 'picked_up',
  ...extra,
})
const route = {
  date: '2026-10-02',
  total_packages: 3,
  stops: [
    { number: 1, address: 'Praça da Sé, 10 - Sé, São Paulo', latitude: -23.55, longitude: -46.63, packages: [pkg(1, 1, 'Maria', '10', { complement: 'Apto 2' }), pkg(2, 2, 'Ana', '10')] },
    { number: 2, address: 'Praça da Sé, 20 - Sé, São Paulo', latitude: null, longitude: null, packages: [pkg(3, 3, 'Carlos', '20')] },
  ],
}

describe('rota do motorista', () => {
  it('mostra as paradas numeradas, com os pacotes de 1 a N', async () => {
    mockApi({ 'POST /auth/refresh': () => [200, session], 'GET /me/route': () => [200, route] })
    renderApp('/motorista/rota')
    expect(await screen.findByText('2 paradas · 3 pacotes')).toBeInTheDocument()
    expect(screen.getByLabelText('Parada 1')).toBeInTheDocument()
    expect(screen.getByText('#2')).toBeInTheDocument()
    expect(screen.getByText('Apto 2')).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Ligar (11) 98765-4321' })[0]).toHaveAttribute('href', 'tel:+5511987654321')
    expect(screen.getByText(/1 parada não tem ponto no mapa/)).toBeInTheDocument()
  })

  it('digitar o código carrega o pacote', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, { date: '2026-10-02', total_packages: 0, stops: [] }],
      'GET /me/deliveries': () => [200, []],
      'POST /me/route/deliveries': () => [200, route],
    })
    renderApp('/motorista/rota')
    expect(await screen.findByText(/Nenhum pacote na rota/)).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Código do pacote'), route.stops[1].packages[0].tracking_code)
    await userEvent.click(screen.getByRole('button', { name: 'Adicionar' }))
    expect(await screen.findByText('Pacote 3 carregado (parada 2): Carlos.')).toBeInTheDocument()
    expect(calls.find((c) => c.method === 'POST' && c.path === '/me/route/deliveries')!.body).toEqual({
      code: route.stops[1].packages[0].tracking_code,
    })
  })

  it('pacote de outro motorista é recusado com o motivo', async () => {
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, route],
      'POST /me/route/deliveries': () => [409, { error: 'conflict: the delivery is assigned to another driver' }],
    })
    renderApp('/motorista/rota')
    await userEvent.type(await screen.findByLabelText('Código do pacote'), 'RSXXXXXXXXXX')
    await userEvent.click(screen.getByRole('button', { name: 'Adicionar' }))
    expect(await screen.findByText('Este pacote é de outro motorista.')).toBeInTheDocument()
  })

  it('descer uma parada manda a ordem nova com os pacotes juntos', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, route],
      'PUT /me/route/order': () => [200, route],
    })
    renderApp('/motorista/rota')
    await userEvent.click(await screen.findByRole('button', { name: 'Descer parada 1' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    expect(calls.find((c) => c.method === 'PUT')!.body).toEqual({ delivery_ids: [3, 1, 2] })
  })

  it('arrastar uma parada para baixo salva a ordem nova', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, route],
      'PUT /me/route/order': () => [200, route],
    })
    renderApp('/motorista/rota')
    const handle = await screen.findByRole('button', { name: 'Arrastar parada 1' })
    // O jsdom não calcula layout nem captura o ponteiro: cada cartão tem 90 px,
    // com 10 px entre eles, na ordem em que está na lista.
    const list = screen.getByRole('list', { name: 'Paradas' })
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      const i = Array.from(list.children).indexOf(this)
      return DOMRect.fromRect({ x: 0, y: i * 100, width: 300, height: 90 })
    })
    HTMLElement.prototype.setPointerCapture = () => {}

    fireEvent.pointerDown(handle, { pointerId: 1, clientY: 20 })
    // O React move o cartão arrastado e o navegador perde a captura: o resto
    // do gesto chega pela janela.
    fireEvent.pointerMove(window, { pointerId: 1, clientY: 80 })
    expect(screen.getAllByRole('button', { name: /^Arrastar parada/ })).toHaveLength(2)
    fireEvent.pointerMove(window, { pointerId: 1, clientY: 120 })
    fireEvent.pointerUp(window, { pointerId: 1, clientY: 120 })

    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    expect(calls.find((c) => c.method === 'PUT')!.body).toEqual({ delivery_ids: [3, 1, 2] })
    expect(screen.getByRole('button', { name: 'Subir parada 2' })).toBeEnabled()
  })

  it('a rota mudou em outro aparelho: recarrega e avisa', async () => {
    let routeCalls = 0
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, routeCalls++ === 0 ? route : { ...route, total_packages: 2, stops: [route.stops[0]] }],
      'PUT /me/route/order': () => [422, { error: 'invalid input', fields: { delivery_ids: 'must list every package of the route once' } }],
    })
    renderApp('/motorista/rota')
    await userEvent.click(await screen.findByRole('button', { name: 'Descer parada 1' }))
    expect(await screen.findByText('A rota mudou. Atualizamos a lista, confira e tente de novo.')).toBeInTheDocument()
    expect(await screen.findByText('1 parada · 2 pacotes')).toBeInTheDocument()
  })

  it('depois de "Entreguei", a rota mostra o status novo', async () => {
    let status = 'in_transit'
    const one = () => pkg(3, 3, 'Carlos', '20', { status })
    mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, { ...route, total_packages: 1, stops: [{ ...route.stops[1], packages: [one()] }] }],
      'GET /deliveries/3': () => [200, one()],
      'GET /deliveries/3/events': () => [200, []],
      'GET /me/deliveries': () => [200, [one()]],
      'POST /deliveries/3/events': () => {
        status = 'delivered'
        return [201, { id: 9, delivery_id: 3, status, note: null, created_by: 2, created_at: '2026-10-01T13:00:00Z' }]
      },
    })
    // Como no main.tsx: sem o aviso, a rota voltaria do cache com o status velho.
    renderApp('/motorista/rota', undefined, { staleTime: 30_000, refetchOnWindowFocus: false })
    await userEvent.click(await screen.findByRole('link', { name: 'Carlos' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Entreguei' }))
    expect(await screen.findByText('Status atualizado: Entregue.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('link', { name: 'Rota de hoje' }))
    expect(await screen.findByText('Entregue')).toBeInTheDocument()
  })

  it('organizar pede a melhor rota', async () => {
    const calls = mockApi({
      'POST /auth/refresh': () => [200, session],
      'GET /me/route': () => [200, route],
      'POST /me/route/optimize': () => [200, route],
    })
    renderApp('/motorista/rota')
    await userEvent.click(await screen.findByRole('button', { name: 'Organizar melhor rota' }))
    expect(await screen.findByText('Rota organizada. Mude a ordem se preferir.')).toBeInTheDocument()
    // Sem localização (como aqui), a ordem começa pela primeira parada.
    expect(calls.find((c) => c.path === '/me/route/optimize')!.body).toEqual({})
  })
})
