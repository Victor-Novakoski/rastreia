/**
 * Content-Security-Policy do front (SECURITY.md #20). Só scripts e estilos do
 * próprio site, nada inline, e conexões só com a API (HTTP e WebSocket) e com
 * os serviços gratuitos de endereço: ViaCEP para o CEP e Nominatim para achar
 * o ponto no mapa. As imagens do mapa vêm do OpenStreetMap.
 */
export function contentSecurityPolicy(apiURL: string): string {
  const api = new URL(apiURL).origin
  // O WebSocket usa o mesmo host com ws(s)://, que nem todo navegador aceita como http(s).
  const live = api.replace(/^http/, 'ws')
  return [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self'",
    "img-src 'self' data: https://tile.openstreetmap.org",
    "font-src 'self'",
    `connect-src 'self' ${api} ${live} https://viacep.com.br https://nominatim.openstreetmap.org`,
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join('; ')
}
