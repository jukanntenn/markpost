export const availableLocales = ["en", "zh"] as const;
export type Locale = (typeof availableLocales)[number];
