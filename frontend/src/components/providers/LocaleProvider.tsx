"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { NextIntlClientProvider, type IntlError } from "next-intl";
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
// with an empty messages object — the first render shows a neutral loading
// state, then the hydration effect loads the user's preferred locale chunk and
// re-renders with real messages. Under static export the root layout cannot
// call getLocale()/getMessages() (server-only), so this self-bootstraps.
//
// Booting with `messages={}` means next-intl raises MISSING_MESSAGE for every
// accessed namespace during that one bootstrap frame. Those errors are pure
// noise (the messages genuinely haven't loaded yet, by design), so onError
// suppresses MISSING_MESSAGE only while messages are still empty. Once a locale
// chunk has loaded, real missing-key errors surface normally. See
// specs/frontend/i18n.md.
export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(defaultLocale);
  const [messages, setMessages] = useState<AbstractIntlMessages>({});

  const messagesLoaded = useRef(false);

  const applyMessages = useCallback(
    (newLocale: Locale, m: AbstractIntlMessages) => {
      setLocaleState(newLocale);
      setMessages(m);
      messagesLoaded.current = true;
      document.documentElement.lang = newLocale;
      setCurrentLocale(newLocale);
    },
    [],
  );

  useEffect(() => {
    const stored = getDefaultLocale();
    loadMessages(stored).then((m) => applyMessages(stored, m));
  }, [applyMessages]);

  const setLocale = useCallback(
    async (newLocale: Locale) => {
      const m = await loadMessages(newLocale);
      applyMessages(newLocale, m);
      persistLocale(newLocale);
    },
    [applyMessages],
  );

  const onError = useCallback((error: IntlError) => {
    if (error.code === "MISSING_MESSAGE" && !messagesLoaded.current) {
      return;
    }
    console.error(error);
  }, []);

  return (
    <LocaleContext.Provider value={{ locale, setLocale, availableLocales }}>
      <NextIntlClientProvider locale={locale} messages={messages} onError={onError}>
        {children}
      </NextIntlClientProvider>
    </LocaleContext.Provider>
  );
}
