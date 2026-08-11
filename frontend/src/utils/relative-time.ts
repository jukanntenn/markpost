import { formatToLocalTime } from './time'

// I.7/J.7 相对时间：以客户端本地现在为基准（Intl.RelativeTimeFormat），
// 绝对时间用 locale 格式。
const rtfCache = new Map<string, Intl.RelativeTimeFormat>()

function getRTF(locale: string): Intl.RelativeTimeFormat {
  let rtf = rtfCache.get(locale)
  if (!rtf) {
    rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
    rtfCache.set(locale, rtf)
  }
  return rtf
}

export function relativeTime(utcString: string, locale: string): string {
  if (!utcString) return ''
  const date = new Date(utcString)
  if (isNaN(date.getTime())) return ''
  const diffMs = date.getTime() - Date.now()
  const abs = Math.abs(diffMs)
  const rtf = getRTF(locale)

  if (abs < 60_000) return rtf.format(Math.round(diffMs / 1000), 'second')
  if (abs < 3_600_000) return rtf.format(Math.round(diffMs / 60_000), 'minute')
  if (abs < 86_400_000)
    return rtf.format(Math.round(diffMs / 3_600_000), 'hour')
  if (abs < 2_592_000_000)
    return rtf.format(Math.round(diffMs / 86_400_000), 'day')
  return formatToLocalTime(utcString, { includeSeconds: false, locale })
}
