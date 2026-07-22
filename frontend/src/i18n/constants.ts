export const availableLocales = ["en", "zh-Hans", "zh-Hant", "ja"] as const;
export type Locale = (typeof availableLocales)[number];
export const defaultLocale: Locale = "en";

// Self-named display labels keyed by locale code so the selector always shows a
// label for every entry in availableLocales.
export const localeNames: Record<Locale, string> = {
  en: "English",
  "zh-Hans": "简体中文",
  "zh-Hant": "繁體中文",
  ja: "日本語",
};
