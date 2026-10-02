/** Marca do Rastreia: um pino de mapa com uma caixa. Decorativo; o nome vem ao lado. */
export function Logo({ className = '' }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" className={className} aria-hidden="true">
      <rect width="32" height="32" rx="8" fill="currentColor" opacity="0.15" />
      <path d="M16 5a8 8 0 0 0-8 8c0 6 8 14 8 14s8-8 8-14a8 8 0 0 0-8-8Z" fill="currentColor" />
      <path d="M12.5 11.5 16 9.5l3.5 2v4L16 17.5l-3.5-2Z" fill="none" stroke="var(--color-brand-800)" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M12.5 11.5 16 13.5l3.5-2M16 13.5v4" fill="none" stroke="var(--color-brand-800)" strokeWidth="1.5" strokeLinejoin="round" />
    </svg>
  )
}
