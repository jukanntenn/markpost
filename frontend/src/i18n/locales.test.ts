import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { IntlMessageFormat } from 'intl-messageformat'

const LOCALES_DIR = path.resolve(__dirname, 'locales')
const LOCALES = ['en', 'zh-Hans', 'zh-Hant', 'ja']

// Flatten nested locale JSON into dot-paths, e.g. "admin.posts.deleteConfirm".
function flattenMessages(obj: Record<string, unknown>, prefix = ''): string[] {
  const keys: string[] = []
  for (const [key, value] of Object.entries(obj)) {
    const pathKey = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      keys.push(...flattenMessages(value as Record<string, unknown>, pathKey))
    } else {
      keys.push(pathKey)
    }
  }
  return keys
}

function loadLocale(locale: string): Record<string, unknown> {
  const file = path.join(LOCALES_DIR, `${locale}.json`)
  return JSON.parse(fs.readFileSync(file, 'utf-8'))
}

describe('locale messages are valid ICU MessageFormat', () => {
  for (const locale of LOCALES) {
    const messages = loadLocale(locale)

    it(`${locale}: all messages parse without MALFORMED_ARGUMENT`, () => {
      const failures: string[] = []
      for (const [dottedKey, raw] of Object.entries(flatten(messages))) {
        try {
          new IntlMessageFormat(raw as string, locale)
        } catch (err) {
          failures.push(`${dottedKey}: ${(err as Error).message}`)
        }
      }
      expect(
        failures,
        `${locale} has malformed ICU messages:\n${failures.join('\n')}`
      ).toEqual([])
    })
  }

  it('all locales share the same set of message keys', () => {
    const keySets = LOCALES.map((locale) => {
      const messages = loadLocale(locale)
      return new Set(flattenMessages(messages))
    })
    const baseline = keySets[0]
    for (let i = 1; i < keySets.length; i++) {
      const missing = [...baseline].filter((k) => !keySets[i].has(k))
      const extra = [...keySets[i]].filter((k) => !baseline.has(k))
      expect(
        missing,
        `${LOCALES[i]} missing keys present in ${LOCALES[0]}: ${missing.join(', ')}`
      ).toEqual([])
      expect(
        extra,
        `${LOCALES[i]} has keys absent in ${LOCALES[0]}: ${extra.join(', ')}`
      ).toEqual([])
    }
  })
})

// Same as flattenMessages but returns the {path: value} map for parsing.
function flatten(
  obj: Record<string, unknown>,
  prefix = ''
): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj)) {
    const pathKey = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      Object.assign(out, flatten(value as Record<string, unknown>, pathKey))
    } else {
      out[pathKey] = value
    }
  }
  return out
}
