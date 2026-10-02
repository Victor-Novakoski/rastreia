import { useState, type FormEvent } from 'react'
import type { DeliveryInput, Driver } from '../lib/deliveries'
import type { FieldErrors } from '../lib/fields'
import { Button } from './Button'
import { SelectField, TextArea, TextField } from './Field'

type Props = {
  initial?: DeliveryInput
  drivers: Driver[]
  errors: FieldErrors
  sending: boolean
  submitLabel: string
  onSubmit: (input: DeliveryInput) => void
}

/** Formulário de criação e edição de entrega. */
export function DeliveryForm({ initial, drivers, errors, sending, submitLabel, onSubmit }: Props) {
  const [name, setName] = useState(initial?.recipient_name ?? '')
  const [email, setEmail] = useState(initial?.recipient_email ?? '')
  const [address, setAddress] = useState(initial?.address ?? '')
  const [driver, setDriver] = useState(initial?.driver_id ? String(initial.driver_id) : '')

  function submit(e: FormEvent) {
    e.preventDefault()
    if (sending) return
    onSubmit({
      recipient_name: name.trim(),
      recipient_email: email.trim(),
      address: address.trim(),
      driver_id: driver ? Number(driver) : undefined,
    })
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-4">
      <TextField
        label="Nome do destinatário"
        value={name}
        onChange={(e) => setName(e.target.value)}
        required
        maxLength={120}
        error={errors.recipient_name}
      />
      <TextField
        label="E-mail do destinatário"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        required
        error={errors.recipient_email}
        hint="Recebe o link de rastreio."
      />
      <TextArea
        label="Endereço"
        value={address}
        onChange={(e) => setAddress(e.target.value)}
        required
        maxLength={300}
        rows={2}
        error={errors.address}
      />
      <SelectField
        label="Motorista"
        value={driver}
        onChange={(e) => setDriver(e.target.value)}
        error={errors.driver_id}
        // A API não tira o motorista de uma entrega, só troca.
        hint={initial?.driver_id ? undefined : 'Pode ficar para depois.'}
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
