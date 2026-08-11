'use client'

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react'
import { NextIntlClientProvider } from 'next-intl'
import type { AbstractIntlMessages } from 'next-intl'
import { availableLocales, defaultLocale, type Locale } from '@/i18n/constants'
import { setCurrentLocale } from '@/i18n/current'
import { setClientMessages } from '@/i18n/messages'
import { getDefaultLocale, loadMessages, persistLocale } from '@/utils/i18n'
import { setDefaultLocale } from '@/utils/time'
import { AppShellSkeleton } from './AppShellSkeleton'

interface LocaleContextValue {
  locale: Locale
  setLocale: (locale: Locale) => void
  availableLocales: readonly Locale[]
}

export const LocaleContext = createContext<LocaleContextValue | null>(null)

export function useLocaleContext() {
  const ctx = useContext(LocaleContext)
  if (!ctx)
    throw new Error('useLocaleContext must be used within LocaleProvider')
  return ctx
}

// LocaleProvider is a pure client-side provider (no server props). Under static
// export the root layout cannot call getLocale()/getMessages() (server-only),
// so this self-bootstraps: the messages chunk is loaded asynchronously after
// hydration.
//
// To avoid a "flash of untranslated keys", the tree is gated behind `ready`:
// until the first locale chunk resolves we render AppShellSkeleton instead of
// children, so no `useTranslations` call runs with empty messages (which would
// otherwise emit the dotted key path via next-intl's message fallback). Once a
// chunk loads, children mount once with real messages. See specs/frontend/i18n.md.
export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(defaultLocale)
  const [messages, setMessages] = useState<AbstractIntlMessages>({})
  const [ready, setReady] = useState(false)

  const applyMessages = useCallback(
    (newLocale: Locale, m: AbstractIntlMessages) => {
      setLocaleState(newLocale)
      setMessages(m)
      setClientMessages(m)
      document.documentElement.lang = newLocale
      setCurrentLocale(newLocale)
      setDefaultLocale(newLocale)
      setReady(true)
    },
    [],
  )

  useEffect(() => {
    const stored = getDefaultLocale()
    loadMessages(stored).then((m) => applyMessages(stored, m))
  }, [applyMessages])

  const setLocale = useCallback(
    async (newLocale: Locale) => {
      const m = await loadMessages(newLocale)
      applyMessages(newLocale, m)
      persistLocale(newLocale)
    },
    [applyMessages],
  )

  return (
    <LocaleContext.Provider value={{ locale, setLocale, availableLocales }}>
      {ready ? (
        <NextIntlClientProvider locale={locale} messages={messages}>
          {children}
        </NextIntlClientProvider>
      ) : (
        <AppShellSkeleton />
      )}
    </LocaleContext.Provider>
  )
}
