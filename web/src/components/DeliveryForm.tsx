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
  const numberInput = useRef<HTMLInputElement>(null)

  async function fillFromCEP(value: string) {
    if (digits(value).length !== 8) return
    setCEPLookup({ state: 'loading' })
    try {
      const found = await lookupCEP(value)
      if (!found) {
        setCEPLookup({ state: 'error', message: 'CEP não encontrado. Preencha o endereço à mão.' })
        return
      }
      // CEP de cidade pequena não tem rua: só preenche o que veio.
      if (found.street) setStreet(found.street)
      if (found.district) setDistrict(found.district)
      setCity(found.city)
      setUF(found.state)
      setPin(null)
      setCEPLookup({ state: 'done' })
      numberInput.current?.focus()
    } catch {
      setCEPLookup({ state: 'error', message: 'Não deu para buscar o CEP agora. Preencha o endereço à mão.' })
    }
  }

  async function findOnMap() {
    setMapLookup({ state: 'loading' })
    try {
      const found = await geocode({ street, number, city, state: uf, postal_code: digits(cep) })
      if (found) {
        setPin(found)
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
            label="CEP"
            inputMode="numeric"
            value={cep}
            onChange={(e) => {
              const v = formatCEP(e.target.value)
              setCEP(v)
              if (digits(v).length === 8 && digits(v) !== digits(cep)) void fillFromCEP(v)
            }}
            required={addressRequired}
            placeholder="01001-000"
            error={errors.postal_code ?? (cepLookup.state === 'error' ? cepLookup.message : undefined)}
            hint={cepLookup.state === 'loading' ? 'Buscando o CEP…' : undefined}
          />
          <TextField
            label="Rua"
            value={street}
            onChange={(e) => setStreet(e.target.value)}
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
            onChange={(e) => setNumber(e.target.value)}
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
            onChange={(e) => setCity(e.target.value)}
            required={addressRequired}
            maxLength={100}
            error={errors.city}
          />
          <SelectField label="UF" value={uf} onChange={(e) => setUF(e.target.value)} required={addressRequired} error={errors.state}>
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
            <button type="button" onClick={() => setPin(null)} className="min-h-11 px-2 font-medium text-brand-700">
              Tirar o pino
            </button>
          )}
        </div>
        {mapLookup.state === 'error' && <p className="text-sm text-danger-fg">{mapLookup.message}</p>}
        {(errors.latitude || errors.longitude) && (
          <p className="text-sm text-danger-fg">Ponto do mapa inválido. Marque de novo.</p>
        )}
        <PinMap value={pin} onChange={setPin} label="Mapa com o local da entrega" />
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
