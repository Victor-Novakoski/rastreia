import { describe, expect, it } from 'vitest'
import { formatDateTime } from './format'
import { isValidCode, normalizeCode } from './tracking'

describe('normalizeCode', () => {
  it('tira espaços e hífens e passa para maiúsculas', () => {
    expect(normalizeCode(' rs7k-2m9q xa4p ')).toBe('RS7K2M9QXA4P')
  })
})

describe('isValidCode', () => {
  it.each([
    ['RS7K2M9QXA4P', true],
    ['RS7K2M9QXA4', false], // curto
    ['XX7K2M9QXA4P', false], // prefixo
    ['RS7K2M9QXA0P', false], // 0 não existe no alfabeto
    ['RS7K2M9QXAIP', false], // I não existe no alfabeto
  ])('%s → %s', (code, want) => {
    expect(isValidCode(code)).toBe(want)
  })
})

describe('formatDateTime', () => {
  it('usa dd/mm/aaaa HH:mm', () => {
    expect(formatDateTime('2026-10-01T20:05:00Z')).toMatch(/^\d{2}\/\d{2}\/\d{4} \d{2}:\d{2}$/)
  })
})
