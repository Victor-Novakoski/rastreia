export const statuses = ['pending', 'picked_up', 'in_transit', 'delivered', 'failed'] as const

export type Status = (typeof statuses)[number]

export type Tone = 'neutral' | 'info' | 'highlight' | 'success' | 'danger'

/** Rótulo e cor semântica de cada status (DESIGN.md, "Status na interface"). */
export const statusInfo: Record<Status, { label: string; tone: Tone }> = {
  pending: { label: 'Aguardando coleta', tone: 'neutral' },
  picked_up: { label: 'Coletado', tone: 'info' },
  in_transit: { label: 'Em rota', tone: 'highlight' },
  delivered: { label: 'Entregue', tone: 'success' },
  failed: { label: 'Não entregue', tone: 'danger' },
}

/** Classes do Tailwind para cada cor semântica (tokens em index.css). */
export const toneClasses: Record<Tone, string> = {
  neutral: 'bg-neutral-bg text-neutral-fg',
  info: 'bg-info-bg text-info-fg',
  highlight: 'bg-highlight-bg text-highlight-fg',
  success: 'bg-success-bg text-success-fg',
  danger: 'bg-danger-bg text-danger-fg',
}
