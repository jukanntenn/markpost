let defaultLocale = 'en'

export function setDefaultLocale(locale: string) {
  defaultLocale = toIntlLocale(locale)
}

// toIntlLocale maps the app's locale identifiers to BCP 47 codes that
// Intl.DateTimeFormat understands. The app uses Unicode locale names
// (zh-Hans/zh-Hant) and a bare "ja", while Intl needs region-qualified tags
// (zh-CN/zh-TW/ja-JP) to render dates in the expected calendar convention.
export function toIntlLocale(locale: string): string {
  switch (locale) {
    case 'zh-Hans':
      return 'zh-CN'
    case 'zh-Hant':
      return 'zh-TW'
    case 'ja':
      return 'ja-JP'
    default:
      return 'en-US'
  }
}

const formatterCache = new Map<string, Intl.DateTimeFormat>()

function getCachedFormatter(
  locale: string,
  options: Intl.DateTimeFormatOptions,
): Intl.DateTimeFormat {
  const key = `${locale}:${JSON.stringify(options)}`
  let formatter = formatterCache.get(key)
  if (!formatter) {
    formatter = new Intl.DateTimeFormat(locale, options)
    formatterCache.set(key, formatter)
  }
  return formatter
}

export interface FormatTimeOptions {
  includeSeconds?: boolean
  locale?: string
}

export function formatToLocalTime(
  utcString: string,
  options?: FormatTimeOptions,
): string {
  if (!utcString) return ''

  const date = new Date(utcString)
  if (isNaN(date.getTime())) return ''

  const includeSeconds = options?.includeSeconds ?? true
  const locale = options?.locale ?? defaultLocale

  const formatter = getCachedFormatter(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    ...(includeSeconds ? { second: '2-digit' } : {}),
  })

  return formatter.format(date)
}
