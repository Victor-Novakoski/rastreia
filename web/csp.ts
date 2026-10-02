/**
 * Content-Security-Policy do front (SECURITY.md #20). Só scripts e estilos do
 * próprio site, nada inline, e conexões só com a API (HTTP e WebSocket).
 */
export function contentSecurityPolicy(apiURL: string): string {
  const api = new URL(apiURL).origin
  // O WebSocket usa o mesmo host com ws(s)://, que nem todo navegador aceita como http(s).
  const live = api.replace(/^http/, 'ws')
  return [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self'",
    "img-src 'self' data:",
    "font-src 'self'",
    `connect-src 'self' ${api} ${live}`,
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join('; ')
}
