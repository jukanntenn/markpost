import { availableLocales, type Locale } from "@/i18n/constants";

const STORAGE_KEY = "locale";

export function getDefaultLocale(): Locale {
  if (typeof window === "undefined") return "en";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored && availableLocales.includes(stored as Locale)) return stored as Locale;
  const browserLang = navigator.language.split("-")[0];
  if (availableLocales.includes(browserLang as Locale)) return browserLang as Locale;
  return "en";
}

export function persistLocale(locale: Locale): void {
  localStorage.setItem(STORAGE_KEY, locale);
  document.cookie = `locale=${locale};path=/;max-age=${60 * 60 * 24 * 365};samesite=lax`;
}

export async function loadMessages(locale: Locale) {
  return (await import(`@/i18n/locales/${locale}.json`)).default;
}
