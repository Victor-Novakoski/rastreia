import { useId, type ComponentProps, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'

const control =
  'min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 focus:border-brand-600 focus:outline-2 focus:outline-brand-600 aria-invalid:border-danger-fg disabled:bg-slate-100'

type Common = { label: string; error?: string; hint?: string }

/** Rótulo, controle e mensagem de erro embaixo do campo (DESIGN.md). */
function Wrapper({ id, label, error, hint, children }: Common & { id: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="font-medium">
        {label}
      </label>
      {children}
      {hint && !error && <p className="text-sm text-slate-600">{hint}</p>}
      {error && (
        <p id={`${id}-error`} className="text-sm text-danger-fg">
          {error}
        </p>
      )}
    </div>
  )
}

function a11y(id: string, error?: string) {
  return { id, 'aria-invalid': error ? true : undefined, 'aria-describedby': error ? `${id}-error` : undefined }
}

export function TextField({ label, error, hint, ...rest }: Common & ComponentProps<'input'>) {
  const id = useId()
  return (
    <Wrapper id={id} label={label} error={error} hint={hint}>
      <input {...rest} {...a11y(id, error)} className={control} />
    </Wrapper>
  )
}

export function TextArea({ label, error, hint, ...rest }: Common & TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const id = useId()
  return (
    <Wrapper id={id} label={label} error={error} hint={hint}>
      <textarea {...rest} {...a11y(id, error)} className={`${control} py-2`} />
    </Wrapper>
  )
}

export function SelectField({ label, error, hint, children, ...rest }: Common & SelectHTMLAttributes<HTMLSelectElement>) {
  const id = useId()
  return (
    <Wrapper id={id} label={label} error={error} hint={hint}>
      <select {...rest} {...a11y(id, error)} className={control}>
        {children}
      </select>
    </Wrapper>
  )
}
