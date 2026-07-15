import { availableLocales, type Locale } from "@/i18n/constants";

const STORAGE_KEY = "locale";

// matchesLocale reports whether a BCP 47 tag matches one of our supported
// locales. Chinese is special-cased: zh-Hans matches zh-CN/zh-SG (Simplified),
// zh-Hant matches zh-TW/zh-HK (Traditional).
function matchesLocale(tag: string, locale: Locale): boolean {
  const lower = tag.toLowerCase();
  if (lower === locale.toLowerCase()) return true;
  if (locale === "zh-Hans") return lower === "zh-cn" || lower === "zh-sg";
  if (locale === "zh-Hant") return lower === "zh-tw" || lower === "zh-hk" || lower === "zh-mo";
  if (locale === "en") return lower.startsWith("en");
  if (locale === "ja") return lower.startsWith("ja");
  return false;
}

export function getDefaultLocale(): Locale {
  if (typeof window === "undefined") return "en";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored && availableLocales.includes(stored as Locale)) return stored as Locale;
  for (const candidate of navigator.languages) {
    for (const locale of availableLocales) {
      if (matchesLocale(candidate, locale)) return locale;
    }
  }
  return "en";
}

// persistLocale stores the preference in localStorage only — the backend reads
// locale from the Accept-Language header, not a cookie (specs/frontend/i18n.md).
export function persistLocale(locale: Locale): void {
  localStorage.setItem(STORAGE_KEY, locale);
}

export async function loadMessages(locale: Locale) {
  return (await import(`@/i18n/locales/${locale}.json`)).default;
}
