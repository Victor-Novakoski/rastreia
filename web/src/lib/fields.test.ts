import { describe, expect, it } from 'vitest'
import { ApiError } from './api'
import { fieldErrors } from './fields'

describe('fieldErrors', () => {
  it('traduz o que a API diz de cada campo do formulário de entrega', () => {
    const err = new ApiError(422, 'validation failed', {
      recipient_phone: 'must have the area code and 10 or 11 digits',
      postal_code: 'must have 8 digits',
      state: 'must be a Brazilian state (UF)',
      street: 'must have at most 200 characters',
      number: 'must have at most 20 characters',
      city: 'must have at most 100 characters',
    })
    expect(fieldErrors(err)).toEqual({
      recipient_phone: 'Use o DDD e o número: 10 ou 11 dígitos.',
      postal_code: 'O CEP tem 8 dígitos.',
      state: 'Use a sigla de um estado, como SP.',
      street: 'Máximo de 200 caracteres.',
      number: 'Máximo de 20 caracteres.',
      city: 'Máximo de 100 caracteres.',
    })
  })

  it('mensagem desconhecida aparece como veio; outros erros não marcam campo', () => {
    expect(fieldErrors(new ApiError(422, 'validation failed', { x: 'is odd' }))).toEqual({ x: 'is odd' })
    expect(fieldErrors(new ApiError(409, 'conflict'))).toEqual({})
    expect(fieldErrors(new Error('boom'))).toEqual({})
  })
})
