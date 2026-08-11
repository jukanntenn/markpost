import { describe, expect, it } from 'vitest'
import { passwordStrength } from './password-strength'

// K.5 强度算法：熵估算 + 长度下限。
describe('passwordStrength', () => {
  it('returns weak for empty or very short passwords', () => {
    expect(passwordStrength('')).toBe('weak')
    expect(passwordStrength('abcdefg')).toBe('weak')
    expect(passwordStrength('aA1!')).toBe('weak')
  })

  it('returns weak below 40 bits even with symbols', () => {
    // 7 个符号字符 = 7*5 = 35 bit
    expect(passwordStrength('!@#$%^&')).toBe('weak')
  })

  it('returns fair between 40 and 60 bits', () => {
    // 10 个小写 = 10*4.7 = 47 bit
    expect(passwordStrength('abcdefghij')).toBe('fair')
    // 9 位混合 = 9*5.7 ≈ 51 bit
    expect(passwordStrength('aB3cD5eF7')).toBe('fair')
  })

  it('returns strong above 60 bits', () => {
    // 12 位混合 + 符号 ≈ 68 bit
    expect(passwordStrength('aB3cD5eF7gH!')).toBe('strong')
    // 16 位小写 = 75 bit
    expect(passwordStrength('abcdefghijklmnop')).toBe('strong')
  })
})
