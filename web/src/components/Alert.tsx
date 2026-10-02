import type { ReactNode } from 'react'

export function Alert({ tone = 'danger', children }: { tone?: 'danger' | 'success'; children: ReactNode }) {
  const cls = tone === 'danger' ? 'bg-danger-bg text-danger-fg' : 'bg-success-bg text-success-fg'
  return (
    <div role={tone === 'danger' ? 'alert' : 'status'} className={`rounded-lg px-3 py-2 text-sm font-medium ${cls}`}>
      {children}
    </div>
  )
}
