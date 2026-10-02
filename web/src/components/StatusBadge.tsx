import { statusInfo, toneClasses, type Status } from '../lib/status'
import { StatusIcon } from './StatusIcon'

/** Status sempre com cor, texto e ícone (DESIGN.md). */
export function StatusBadge({ status, size = 'md' }: { status: Status; size?: 'md' | 'lg' }) {
  const { label, tone } = statusInfo[status]
  const sizing = size === 'lg' ? 'gap-2 px-4 py-2 text-lg' : 'gap-1.5 px-2.5 py-1 text-sm'
  return (
    <span className={`inline-flex items-center rounded-full font-semibold ${sizing} ${toneClasses[tone]}`}>
      <StatusIcon status={status} className={size === 'lg' ? 'size-6' : 'size-4'} />
      {label}
    </span>
  )
}
