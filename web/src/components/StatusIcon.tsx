import type { Status } from '../lib/status'

const paths: Record<Status, string> = {
  // relógio
  pending: 'M12 7v5l3 2m6-2a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z',
  // caixa
  picked_up: 'M21 8 12 3 3 8m18 0-9 5m9-5v8l-9 5m0-8L3 8m9 5v8M3 8v8l9 5',
  // caminhão
  in_transit:
    'M3 6h11v10H3zM14 9h4l3 3v4h-7M7.5 18.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Zm11 0a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Z',
  // check
  delivered: 'm5 12.5 4.5 4.5L19 7.5',
  // x
  failed: 'M6 6l12 12M18 6 6 18',
}

export function StatusIcon({ status, className = 'size-5' }: { status: Status; className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d={paths[status]} />
    </svg>
  )
}
