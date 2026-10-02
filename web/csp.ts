/**
 * Content-Security-Policy do front (SECURITY.md #20). Só scripts e estilos do
 * próprio site, nada inline, e conexões só com a API.
 */
export function contentSecurityPolicy(apiURL: string): string {
  const api = new URL(apiURL).origin
  return [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self'",
    "img-src 'self' data:",
    "font-src 'self'",
    `connect-src 'self' ${api}`,
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join('; ')
}
