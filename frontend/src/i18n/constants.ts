export const availableLocales = ["en", "zh-Hans", "zh-Hant", "ja"] as const;
export type Locale = (typeof availableLocales)[number];
export const defaultLocale: Locale = "en";
