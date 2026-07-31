import { describe, expect, it } from 'vitest'
import { formatToLocalTime, setDefaultLocale, toIntlLocale } from './time'

describe('formatToLocalTime', () => {
  it('formats a UTC ISO string with seconds by default', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z')
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/)
  })

  it('formats without seconds when includeSeconds is false', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z', {
      includeSeconds: false,
    })
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2} [AP]M$/)
    const timePart = result.split(', ')[1]
    expect(timePart.split(':')).toHaveLength(2)
  })

  it('defaults to en locale', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z')
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/)
  })

  it('uses the provided locale', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z', {
      locale: 'en-US',
    })
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/)
  })

  it('defaults to en when locale is not provided in object form', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z', {
      includeSeconds: true,
    })
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/)
  })

  it('returns empty string for empty input', () => {
    expect(formatToLocalTime('')).toBe('')
  })

  it('zero-pads single-digit month, day, hour, minute, second', () => {
    const result = formatToLocalTime('2025-01-05T09:05:03Z')
    const [datePart, timePart] = result.split(', ')
    expect(datePart).toMatch(/^\d{2}\/\d{2}\/\d{4}$/)
    expect(timePart).toMatch(/^\d{2}:\d{2}:\d{2} [AP]M$/)
  })

  it('returns empty string for malformed input', () => {
    expect(formatToLocalTime('not-a-date')).toBe('')
  })

  it('handles a date at midnight local time', () => {
    const date = new Date()
    date.setHours(0, 0, 0, 0)
    const result = formatToLocalTime(date.toISOString())
    const timePart = result.split(', ')[1]
    expect(timePart).toBe('12:00:00 AM')
  })

  it('handles includeSeconds as object', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z', {
      includeSeconds: true,
    })
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/)
  })
})

describe('toIntlLocale', () => {
  it('maps zh-Hans to zh-CN', () => {
    expect(toIntlLocale('zh-Hans')).toBe('zh-CN')
  })

  it('maps zh-Hant to zh-TW', () => {
    expect(toIntlLocale('zh-Hant')).toBe('zh-TW')
  })

  it('maps ja to ja-JP', () => {
    expect(toIntlLocale('ja')).toBe('ja-JP')
  })

  it('falls back to en-US for unknown locales', () => {
    expect(toIntlLocale('unknown')).toBe('en-US')
  })
})

describe('formatToLocalTime with non-en locales', () => {
  // zh-CN and other CJK locales use 24-hour time by default, so a midnight UTC
  // instant never reads as "12:xx AM" — the exact confusion that motivated this.
  it('parses a +00:00 offset identically to a Z suffix', () => {
    const withZ = formatToLocalTime('2025-06-15T14:30:45Z', {
      locale: 'zh-CN',
    })
    const withOffset = formatToLocalTime('2025-06-15T14:30:45+00:00', {
      locale: 'zh-CN',
    })
    expect(withOffset).toBe(withZ)
  })

  it('uses 24-hour format under zh-CN (no AM/PM marker)', () => {
    const result = formatToLocalTime('2025-06-15T14:30:45Z', {
      locale: 'zh-CN',
    })
    expect(result).not.toMatch(/[AP]M/)
    expect(result).toMatch(/\d{2}:\d{2}:\d{2}/)
  })

  it('honors setDefaultLocale for the global default', () => {
    setDefaultLocale('zh-Hans')
    const result = formatToLocalTime('2025-06-15T14:30:45Z')
    expect(result).not.toMatch(/[AP]M/)
    // restore for other test suites sharing module state
    setDefaultLocale('en')
  })
})
