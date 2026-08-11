import { describe, expect, it } from 'vitest'
import { safeNext } from './safe-next'

// K.3 intended-URL 安全：防 open redirect。
describe('safeNext', () => {
  it('falls back to /dashboard for null/undefined', () => {
    expect(safeNext(null)).toBe('/dashboard')
    expect(safeNext(undefined)).toBe('/dashboard')
  })

  it('rejects protocol-relative and external URLs', () => {
    expect(safeNext('//evil.com')).toBe('/dashboard')
    expect(safeNext('https://evil.com')).toBe('/dashboard')
    expect(safeNext('http://evil.com/phish')).toBe('/dashboard')
    expect(safeNext('javascript:alert(1)')).toBe('/dashboard')
  })

  it('accepts internal absolute paths', () => {
    expect(safeNext('/posts')).toBe('/posts')
    expect(safeNext('/delivery/history')).toBe('/delivery/history')
  })
})
