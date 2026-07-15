"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { NextIntlClientProvider } from "next-intl";
import type { AbstractIntlMessages } from "next-intl";
import { availableLocales, defaultLocale, type Locale } from "@/i18n/constants";
import { setCurrentLocale } from "@/i18n/current";
import { getDefaultLocale, loadMessages, persistLocale } from "@/utils/i18n";

interface LocaleContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  availableLocales: readonly Locale[];
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function useLocaleContext() {
  const ctx = useContext(LocaleContext);
  if (!ctx) throw new Error("useLocaleContext must be used within LocaleProvider");
  return ctx;
}

// LocaleProvider is a pure client-side provider (no server props). It boots
// with the default locale (en) and, after hydration, reads the stored/browser
// preference and dynamically loads the matching messages chunk. Under static
// export the root layout cannot call getLocale()/getMessages() (server-only),
// so this self-bootstraps instead. See specs/frontend/i18n.md.
export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(defaultLocale);
  const [messages, setMessages] = useState<AbstractIntlMessages>({});

  useEffect(() => {
    const stored = getDefaultLocale();
    loadMessages(stored).then((m) => {
      setLocaleState(stored);
      setMessages(m);
      document.documentElement.lang = stored;
      setCurrentLocale(stored);
    });
  }, []);

  const setLocale = useCallback(async (newLocale: Locale) => {
    const m = await loadMessages(newLocale);
    setLocaleState(newLocale);
    setMessages(m);
    persistLocale(newLocale);
    document.documentElement.lang = newLocale;
    setCurrentLocale(newLocale);
  }, []);

  return (
    <LocaleContext.Provider value={{ locale, setLocale, availableLocales }}>
      <NextIntlClientProvider locale={locale} messages={messages}>
        {children}
      </NextIntlClientProvider>
    </LocaleContext.Provider>
  );
}
