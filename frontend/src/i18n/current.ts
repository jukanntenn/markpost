import { defaultLocale, type Locale } from "./constants";

// A module-level slot holding the current locale, written by LocaleProvider on
// the client. This lets the non-React API client (src/lib/api/base.ts) read the
// current locale to send Accept-Language on every request without threading
// React context through fetch call sites. Defaults to "en" until hydrated.
let currentLocale: Locale = defaultLocale;

export function setCurrentLocale(locale: Locale): void {
  currentLocale = locale;
}

export function getCurrentLocale(): Locale {
  return currentLocale;
}
