import { getRequestConfig } from "next-intl/server";
import { cookies } from "next/headers";
import { availableLocales, type Locale } from "./constants";

const defaultLocale: Locale = "en";

export default getRequestConfig(async () => {
  let locale: Locale = defaultLocale;

  try {
    const cookieStore = await cookies();
    const stored = cookieStore.get("locale")?.value;
    if (stored && availableLocales.includes(stored as Locale)) {
      locale = stored as Locale;
    }
  } catch {
    // cookies() unavailable during static generation
  }

  return {
    locale,
    messages: (await import(`./locales/${locale}.json`)).default,
  };
});
