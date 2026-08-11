import type { AbstractIntlMessages } from 'next-intl'

// A module-level slot holding the loaded message bundles, written by
// LocaleProvider on the client. Lets the non-React API client (base.ts)
// resolve network.* / error messages for frontend-constructed errors
// (network_error / timeout / parse_error, B1.9) without threading context.
let messages: AbstractIntlMessages = {}

export function setClientMessages(m: AbstractIntlMessages): void {
  messages = m
}

function resolvePath(obj: unknown, path: string): unknown {
  let cur: unknown = obj
  for (const key of path.split('.')) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[key]
  }
  return cur
}

// tClient resolves a dotted message key with simple {param} substitution.
// Falls back to the key itself when missing (never throws).
export function tClient(
  key: string,
  params?: Record<string, string | number>,
): string {
  const raw = resolvePath(messages, key)
  if (typeof raw !== 'string') return key
  if (!params) return raw
  return raw.replace(/\{(\w+)\}/g, (m, name: string) =>
    name in params ? String(params[name]) : m,
  )
}
