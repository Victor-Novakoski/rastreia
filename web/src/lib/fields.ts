import { ApiError } from './api'

// Mensagens de validação da API (em inglês) traduzidas para a tela.
const messages: Record<string, string> = {
  'is required': 'Obrigatório.',
  'cannot be empty': 'Não pode ficar vazio.',
  'must be a valid e-mail': 'E-mail inválido.',
  'must have at most 120 characters': 'Máximo de 120 caracteres.',
  'must have at most 300 characters': 'Máximo de 300 caracteres.',
  'must have at most 500 characters': 'Máximo de 500 caracteres.',
  'must have at least 10 characters': 'Mínimo de 10 caracteres.',
  'must have at most 72 bytes': 'Senha longa demais.',
  'is too common': 'Senha muito comum. Escolha outra.',
  'must not contain the e-mail': 'A senha não pode conter o e-mail.',
  'is required when the delivery fails': 'Conte o motivo da falha.',
  'must be an existing driver': 'Escolha um motorista da lista.',
  'must be a valid CNPJ': 'CNPJ inválido. Confira os números.',
}

export type FieldErrors = Record<string, string>

/** Erros de campo de um 422, já em português. Vazio para qualquer outro erro. */
export function fieldErrors(err: unknown): FieldErrors {
  if (!(err instanceof ApiError) || err.status !== 422) return {}
  return Object.fromEntries(Object.entries(err.fields).map(([k, v]) => [k, messages[v] ?? v]))
}
