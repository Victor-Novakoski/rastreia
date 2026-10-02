import { useState, type FormEvent } from 'react'
import { nextStatuses, statusInfo, type Status } from '../lib/status'
import type { FieldErrors } from '../lib/fields'
import { Button } from './Button'
import { SelectField, TextArea } from './Field'

type Props = {
  current: Status
  errors: FieldErrors
  sending: boolean
  onSubmit: (input: { status: Status; note?: string }) => void
}

/** Troca de status: só oferece as transições que a API aceita. */
export function EventForm({ current, errors, sending, onSubmit }: Props) {
  const options = nextStatuses[current]
  const [status, setStatus] = useState<Status | ''>('')
  const [note, setNote] = useState('')
  const chosen = status && options.includes(status) ? status : ''

  if (options.length === 0) return <p className="text-slate-600">Entrega concluída. O status não muda mais.</p>

  function submit(e: FormEvent) {
    e.preventDefault()
    if (sending || !chosen) return
    onSubmit({ status: chosen, note: note.trim() || undefined })
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-4">
      <SelectField
        label="Novo status"
        value={chosen}
        onChange={(e) => setStatus(e.target.value as Status)}
        required
        error={errors.status}
      >
        <option value="" disabled>
          Escolha
        </option>
        {options.map((s) => (
          <option key={s} value={s}>
            {statusInfo[s].label}
          </option>
        ))}
      </SelectField>
      <TextArea
        label={chosen === 'failed' ? 'Motivo (obrigatório)' : 'Observação'}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        required={chosen === 'failed'}
        maxLength={500}
        rows={2}
        error={errors.note}
        hint="Não aparece no rastreio público."
      />
      <Button type="submit" loading={sending} disabled={!chosen} className="self-start">
        Registrar
      </Button>
    </form>
  )
}
