import { useRef, useState, type FormEvent } from 'react'
import { digits, formatCEP, formatPhone, geocode, lookupCEP, states, type LatLng } from '../lib/address'
import type { DeliveryInput, Driver } from '../lib/deliveries'
import type { FieldErrors } from '../lib/fields'
import { Button } from './Button'
import { SelectField, TextArea, TextField } from './Field'
import { PinMap } from './LazyMap'

type Props = {
  initial?: DeliveryInput
  /** Endereço numa linha de uma entrega antiga, que ainda não tem as partes. */
  legacyAddress?: string
  drivers: Driver[]
  errors: FieldErrors
  sending: boolean
  submitLabel: string
  onSubmit: (input: DeliveryInput) => void
}

type Lookup = { state: 'idle' | 'loading' | 'done' } | { state: 'error'; message: string }

/**
 * Formulário de criação e edição de entrega. O CEP preenche rua, bairro,
 * cidade e UF; o endereço vira um pino no mapa que dá para arrastar.
 */
export function DeliveryForm({ initial, legacyAddress, drivers, errors, sending, submitLabel, onSubmit }: Props) {
  const [name, setName] = useState(initial?.recipient_name ?? '')
  const [email, setEmail] = useState(initial?.recipient_email ?? '')
  const [phone, setPhone] = useState(formatPhone(initial?.recipient_phone ?? ''))
  const [cep, setCEP] = useState(formatCEP(initial?.postal_code ?? ''))
  const [street, setStreet] = useState(initial?.street ?? '')
  const [number, setNumber] = useState(initial?.number ?? '')
  const [complement, setComplement] = useState(initial?.complement ?? '')
  const [district, setDistrict] = useState(initial?.district ?? '')
  const [city, setCity] = useState(initial?.city ?? '')
  const [uf, setUF] = useState(initial?.state ?? '')
  const [reference, setReference] = useState(initial?.address_reference ?? '')
  const [pin, setPin] = useState<LatLng | null>(
    initial?.latitude != null && initial.longitude != null ? { latitude: initial.latitude, longitude: initial.longitude } : null,
  )
  const [driver, setDriver] = useState(initial?.driver_id ? String(initial.driver_id) : '')
  const [cepLookup, setCEPLookup] = useState<Lookup>({ state: 'idle' })
  const [mapLookup, setMapLookup] = useState<Lookup>({ state: 'idle' })
  // O pino saiu porque o endereço mudou.
  const [pinMoved, setPinMoved] = useState(false)
  const numberInput = useRef<HTMLInputElement>(null)
  const cepInput = useRef<HTMLInputElement>(null)
  // Só a resposta do último CEP digitado vale.
  const cepRequest = useRef(0)

  /**
   * O pino marca o endereço: se CEP, rua, número, cidade ou UF mudam, ele sai
   * até ser marcado de novo, senão a rota levaria o motorista ao lugar antigo.
   */
  function addressChanged() {
    if (pin) setPinMoved(true)
    setPin(null)
  }

  function placePin(p: LatLng | null) {
    setPin(p)
    setPinMoved(false)
  }

  async function fillFromCEP(value: string) {
    const request = ++cepRequest.current
    setCEPLookup({ state: 'loading' })
    try {
      const found = await lookupCEP(value)
      if (request !== cepRequest.current) return
      if (!found) {
        setCEPLookup({ state: 'error', message: 'CEP não encontrado. Preencha o endereço à mão.' })
        return
      }
      // CEP de cidade pequena não tem rua: só preenche o que veio.
      if (found.street) setStreet(found.street)
      if (found.district) setDistrict(found.district)
      setCity(found.city)
      setUF(found.state)
      setCEPLookup({ state: 'done' })
      // Só leva ao número quem ainda está no CEP, não quem já foi digitar a rua.
      if (document.activeElement === cepInput.current) numberInput.current?.focus()
    } catch {
      if (request !== cepRequest.current) return
      setCEPLookup({ state: 'error', message: 'Não deu para buscar o CEP agora. Preencha o endereço à mão.' })
    }
  }

  function changeCEP(value: string) {
    const v = formatCEP(value)
    setCEP(v)
    if (digits(v) === digits(cep)) return
    addressChanged()
    if (digits(v).length === 8) {
      void fillFromCEP(v)
    } else {
      cepRequest.current++ // a resposta de um CEP anterior não vale mais
      setCEPLookup({ state: 'idle' })
    }
  }

  async function findOnMap() {
    setMapLookup({ state: 'loading' })
    try {
      const found = await geocode({ street, number, city, state: uf, postal_code: digits(cep) })
      if (found) {
        placePin(found)
        setMapLookup({ state: 'done' })
      } else {
        setMapLookup({ state: 'error', message: 'Endereço não achado no mapa. Toque no mapa para marcar o lugar.' })
      }
    } catch {
      setMapLookup({ state: 'error', message: 'Não deu para buscar no mapa agora. Toque no mapa para marcar o lugar.' })
    }
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    if (sending) return
    onSubmit({
      recipient_name: name.trim(),
      recipient_email: email.trim(),
      recipient_phone: digits(phone),
      postal_code: digits(cep),
      street: street.trim(),
      number: number.trim(),
      complement: complement.trim(),
      district: district.trim(),
      city: city.trim(),
      state: uf,
      address_reference: reference.trim(),
      latitude: pin?.latitude ?? null,
      longitude: pin?.longitude ?? null,
      driver_id: driver ? Number(driver) : undefined,
    })
  }

  // Entrega antiga pode mudar outra coisa sem ter de preencher o endereço;
  // se começar a preencher, vai tudo (a API confere o endereço inteiro).
  const addressRequired = !legacyAddress || [cep, street, number, district, city, uf].some((v) => v.trim() !== '')
  const canSearchMap = street.trim() !== '' && city.trim() !== '' && uf !== ''

  return (
    <form onSubmit={submit} className="flex flex-col gap-6">
      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 text-lg font-semibold">Destinatário</legend>
        <TextField
          label="Nome"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          maxLength={120}
          autoComplete="off"
          error={errors.recipient_name}
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField
            label="Telefone"
            type="tel"
            inputMode="tel"
            value={phone}
            onChange={(e) => setPhone(formatPhone(e.target.value))}
            required={addressRequired}
            placeholder="(11) 98765-4321"
            error={errors.recipient_phone}
            hint="Com DDD. O motorista liga se precisar."
          />
          <TextField
            label="E-mail"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            error={errors.recipient_email}
            hint="Recebe o link de rastreio."
          />
        </div>
      </fieldset>

      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 text-lg font-semibold">Endereço</legend>
        {legacyAddress && (
          <p className="rounded-lg bg-neutral-bg p-3 text-sm text-neutral-fg">
            Endereço atual: {legacyAddress}. Preencha as partes abaixo para atualizar.
          </p>
        )}
        <div className="grid gap-4 sm:grid-cols-[10rem_1fr]">
          <TextField
            ref={cepInput}
            label="CEP"
            inputMode="numeric"
            value={cep}
            onChange={(e) => changeCEP(e.target.value)}
            required={addressRequired}
            placeholder="01001-000"
            error={errors.postal_code ?? (cepLookup.state === 'error' ? cepLookup.message : undefined)}
            hint={cepLookup.state === 'loading' ? 'Buscando o CEP…' : undefined}
          />
          <TextField
            label="Rua"
            value={street}
            onChange={(e) => {
              setStreet(e.target.value)
              addressChanged()
            }}
            required={addressRequired}
            maxLength={200}
            error={errors.street}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-[10rem_1fr]">
          <TextField
            ref={numberInput}
            label="Número"
            value={number}
            onChange={(e) => {
              setNumber(e.target.value)
              addressChanged()
            }}
            required={addressRequired}
            maxLength={20}
            error={errors.number}
            hint="Sem número? Escreva S/N."
          />
          <TextField
            label="Complemento"
            value={complement}
            onChange={(e) => setComplement(e.target.value)}
            maxLength={100}
            placeholder="Apto, bloco, fundos"
            error={errors.complement}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-[1fr_1fr_6rem]">
          <TextField
            label="Bairro"
            value={district}
            onChange={(e) => setDistrict(e.target.value)}
            required={addressRequired}
            maxLength={100}
            error={errors.district}
          />
          <TextField
            label="Cidade"
            value={city}
            onChange={(e) => {
              setCity(e.target.value)
              addressChanged()
            }}
            required={addressRequired}
            maxLength={100}
            error={errors.city}
          />
          <SelectField
            label="UF"
            value={uf}
            onChange={(e) => {
              setUF(e.target.value)
              addressChanged()
            }}
            required={addressRequired}
            error={errors.state}
          >
            <option value="" />
            {states.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </SelectField>
        </div>
        <TextArea
          label="Ponto de referência"
          value={reference}
          onChange={(e) => setReference(e.target.value)}
          maxLength={300}
          rows={2}
          placeholder="Portão azul, ao lado da padaria"
          error={errors.address_reference}
          hint="Aparece para o motorista."
        />
      </fieldset>

      <fieldset className="flex flex-col gap-3">
        <legend className="mb-2 text-lg font-semibold">Local no mapa</legend>
        <p className="text-sm text-slate-600">
          Usado para montar a rota do motorista. Se o pino cair no lugar errado, arraste ou toque no mapa.
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            variant="secondary"
            onClick={() => void findOnMap()}
            loading={mapLookup.state === 'loading'}
            disabled={!canSearchMap}
          >
            {pin ? 'Buscar de novo' : 'Achar no mapa'}
          </Button>
          {pin && (
            <button type="button" onClick={() => placePin(null)} className="min-h-11 px-2 font-medium text-brand-700">
              Tirar o pino
            </button>
          )}
        </div>
        {pinMoved && !pin && (
          <p role="status" className="text-sm text-slate-700">
            O endereço mudou, então o pino saiu do mapa. Ache de novo ou toque no mapa.
          </p>
        )}
        {mapLookup.state === 'error' && (
          <p role="alert" className="text-sm text-danger-fg">
            {mapLookup.message}
          </p>
        )}
        {(errors.latitude || errors.longitude) && (
          <p role="alert" className="text-sm text-danger-fg">
            Ponto do mapa inválido. Marque de novo.
          </p>
        )}
        <PinMap value={pin} onChange={placePin} label="Mapa com o local da entrega" />
      </fieldset>

      <SelectField
        label="Motorista"
        value={driver}
        onChange={(e) => setDriver(e.target.value)}
        error={errors.driver_id}
        // A API não tira o motorista de uma entrega, só troca.
        hint={initial?.driver_id ? undefined : 'Pode ficar para depois: o motorista também pega o pacote bipando a etiqueta.'}
      >
        {!initial?.driver_id && <option value="">Sem motorista</option>}
        {drivers.map((d) => (
          <option key={d.id} value={d.id}>
            {d.name}
          </option>
        ))}
      </SelectField>
      <Button type="submit" loading={sending} className="self-start">
        {submitLabel}
      </Button>
    </form>
  )
}
