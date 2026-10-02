/** Aviso de que a tela se atualiza sozinha, mostrado enquanto o WebSocket está aberto. */
export function LiveBadge({ tone = 'light' }: { tone?: 'light' | 'dark' }) {
  const colors = tone === 'dark' ? 'bg-white/10 text-white' : 'bg-emerald-50 text-emerald-800'
  return (
    <span
      role="status"
      title="Esta tela se atualiza sozinha"
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${colors}`}
    >
      <span aria-hidden="true" className="size-2 rounded-full bg-emerald-500" />
      Ao vivo
    </span>
  )
}
