// K.5 强度指示器算法：字符集大小估算熵（bit），
// 弱 <40 bit 或长度<8；中 40-60；强 >60。仅提示不阻断。
export type PasswordStrength = 'weak' | 'fair' | 'strong'

export function passwordStrength(pw: string): PasswordStrength {
  let cs = 0
  if (/[a-z]/.test(pw)) cs += 26
  if (/[A-Z]/.test(pw)) cs += 26
  if (/[0-9]/.test(pw)) cs += 10
  if (/[^a-zA-Z0-9]/.test(pw)) cs += 32
  const entropy = pw.length * Math.log2(cs || 1)
  if (entropy < 40 || pw.length < 8) return 'weak'
  if (entropy < 60) return 'fair'
  return 'strong'
}
