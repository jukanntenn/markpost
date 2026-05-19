let defaultLocale = "en";

export function setDefaultLocale(locale: string) {
  defaultLocale = locale;
}

const formatterCache = new Map<string, Intl.DateTimeFormat>();

function getCachedFormatter(locale: string, options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  const key = `${locale}:${JSON.stringify(options)}`;
  let formatter = formatterCache.get(key);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat(locale, options);
    formatterCache.set(key, formatter);
  }
  return formatter;
}

export interface FormatTimeOptions {
  includeSeconds?: boolean;
  locale?: string;
}

export function formatToLocalTime(utcString: string, options?: FormatTimeOptions): string {
  if (!utcString) return "";

  const date = new Date(utcString);
  if (isNaN(date.getTime())) return "";

  const includeSeconds = options?.includeSeconds ?? true;
  const locale = options?.locale ?? defaultLocale;

  const formatter = getCachedFormatter(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    ...(includeSeconds ? { second: "2-digit" } : {}),
  });

  return formatter.format(date);
}
