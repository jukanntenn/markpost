import { getRequestConfig } from "next-intl/server";

export default getRequestConfig(async () => {
  // Provide a static locale for now
  const locale = "en";

  return {
    locale,
    messages: (await import(`./locales/${locale}.json`)).default,
  };
});
