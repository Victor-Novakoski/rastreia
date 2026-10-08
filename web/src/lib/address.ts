// Serviços gratuitos de endereço, chamados direto do navegador da
// transportadora, sem chave: ViaCEP para o CEP e o Nominatim do OpenStreetMap
// para achar o ponto no mapa. Só o endereço sai daqui, nunca o destinatário.

export const states = [
  'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO', 'MA', 'MT', 'MS', 'MG', 'PA',
  'PB', 'PR', 'PE', 'PI', 'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO',
] as const

export function digits(s: string): string {
  return s.replace(/\D/g, '')
}

/** 01001000 → 01001-000, enquanto a pessoa digita. */
export function formatCEP(s: string): string {
  const d = digits(s).slice(0, 8)
  return d.length > 5 ? `${d.slice(0, 5)}-${d.slice(5)}` : d
}

/** 11987654321 → (11) 98765-4321, enquanto a pessoa digita. */
export function formatPhone(s: string): string {
  const d = digits(s).slice(0, 11)
  if (d.length <= 2) return d
  const local = d.slice(2)
  const split = local.length > 8 ? 5 : 4
  return local.length > split ? `(${d.slice(0, 2)}) ${local.slice(0, split)}-${local.slice(split)}` : `(${d.slice(0, 2)}) ${local}`
}

export type CEPResult = { street: string; district: string; city: string; state: string }

/** Rua, bairro, cidade e UF do CEP; null quando o CEP não existe. */
export async function lookupCEP(cep: string, signal?: AbortSignal): Promise<CEPResult | null> {
  const res = await fetch(`https://viacep.com.br/ws/${digits(cep)}/json/`, { signal })
  if (!res.ok) return null
  const body = (await res.json()) as {
    erro?: boolean | string
    logradouro?: string
    bairro?: string
    localidade?: string
    uf?: string
  }
  if (body.erro) return null
  return { street: body.logradouro ?? '', district: body.bairro ?? '', city: body.localidade ?? '', state: body.uf ?? '' }
}

export type LatLng = { latitude: number; longitude: number }

// O Nominatim pede no máximo uma busca por segundo por usuário.
let lastGeocode = 0

/** Ponto no mapa do endereço, ou null quando o Nominatim não acha. */
export async function geocode(
  a: { street: string; number: string; city: string; state: string; postal_code: string },
  signal?: AbortSignal,
): Promise<LatLng | null> {
  const q = new URLSearchParams({
    format: 'jsonv2',
    limit: '1',
    countrycodes: 'br',
    street: `${a.number} ${a.street}`.trim(),
    city: a.city,
    state: a.state,
  })
  let found = await search(q, signal)
  if (!found && a.postal_code) {
    // Rua nova ou escrita diferente: o centro do CEP já ajuda.
    found = await search(new URLSearchParams({ format: 'jsonv2', limit: '1', countrycodes: 'br', postalcode: formatCEP(a.postal_code) }), signal)
  }
  return found
}

async function search(q: URLSearchParams, signal?: AbortSignal): Promise<LatLng | null> {
  const wait = lastGeocode + 1000 - Date.now()
  if (wait > 0) await new Promise((r) => setTimeout(r, wait))
  lastGeocode = Date.now()
  const res = await fetch(`https://nominatim.openstreetmap.org/search?${q}`, { signal })
  if (!res.ok) return null
  const list = (await res.json()) as { lat: string; lon: string }[]
  if (list.length === 0) return null
  return { latitude: Number(list[0].lat), longitude: Number(list[0].lon) }
}
