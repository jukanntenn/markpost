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
import { availableLocales, type Locale } from "@/i18n/constants";
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

export function LocaleProvider({
  children,
  serverLocale,
  serverMessages,
}: {
  children: React.ReactNode;
  serverLocale: string;
  serverMessages: AbstractIntlMessages;
}) {
  const [locale, setLocaleState] = useState<Locale>(
    availableLocales.includes(serverLocale as Locale)
      ? (serverLocale as Locale)
      : "en",
  );
  const [messages, setMessages] = useState(serverMessages);

  useEffect(() => {
    const stored = getDefaultLocale();
    if (stored !== locale) {
      loadMessages(stored).then((m) => {
        setLocaleState(stored);
        setMessages(m);
      });
    }
  }, [locale]);

  const setLocale = useCallback(async (newLocale: Locale) => {
    const m = await loadMessages(newLocale);
    setLocaleState(newLocale);
    setMessages(m);
    persistLocale(newLocale);
    document.documentElement.lang = newLocale;
  }, []);

  return (
    <LocaleContext.Provider value={{ locale, setLocale, availableLocales }}>
      <NextIntlClientProvider locale={locale} messages={messages}>
        {children}
      </NextIntlClientProvider>
    </LocaleContext.Provider>
  );
}
