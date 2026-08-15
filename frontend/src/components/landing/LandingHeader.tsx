'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import Image from 'next/image'
import { useTranslations } from 'next-intl'
import { LanguagesIcon } from 'lucide-react'
import { useAuthReady } from '@/hooks/useAuthReady'
import { localeNames, type Locale } from '@/i18n/constants'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Button, buttonClass } from '@/components/ui/button'
import { Menu } from '@/components/ui/menu'
import { LANDING_CONTAINER, REPO_URL, DOCS_URL } from './links'

// §00 Masthead：复用应用壳的铬件（56px 高、滚动后 1px hairline + backdrop-blur，
// design.md Elevation 的明文规定）。唯一的主 CTA 留给 Hero，这里用描边样式。
export function LandingHeader() {
  const t = useTranslations('landing.nav')
  const { locale, setLocale, availableLocales } = useLocaleContext()
  const { hasHydrated, isAuthenticated } = useAuthReady()
  const [scrolled, setScrolled] = useState(false)

  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 0)
    window.addEventListener('scroll', handleScroll, { passive: true })
    handleScroll()
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const authed = hasHydrated && isAuthenticated

  return (
    <header
      className={`sticky top-0 z-50 h-(--header-height) w-full bg-background/80 backdrop-blur transition-[border-color] duration-150 ${scrolled ? 'border-b' : ''}`}
    >
      <div className={`flex h-full items-center gap-2 ${LANDING_CONTAINER}`}>
        <Link
          href="/"
          aria-label="markpost"
          className="flex h-11 items-center gap-2.5 rounded-md px-1 transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
        >
          <Image
            src="/markpost.svg"
            alt=""
            className="h-6 w-auto"
            width={24}
            height={24}
          />
          <span className="hidden font-display text-lg font-bold tracking-tight sm:inline">
            markpost
          </span>
        </Link>

        <nav
          aria-label="markpost"
          className="ml-4 hidden items-center gap-1 md:flex"
        >
          <a
            href={DOCS_URL}
            target="_blank"
            rel="noreferrer"
            className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            {t('docs')}
          </a>
          <a
            href={REPO_URL}
            target="_blank"
            rel="noreferrer"
            className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            {t('github')}
          </a>
        </nav>

        <div className="flex-1" />
        <ThemeToggle />
        <Menu.Root>
          <Menu.Trigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-lg"
                aria-label={t('language')}
                title={t('language')}
                className="size-11"
              />
            }
          >
            <LanguagesIcon className="size-4" />
          </Menu.Trigger>
          <Menu.Popup>
            <Menu.RadioGroup
              value={locale}
              onValueChange={(value) => setLocale(value as Locale)}
            >
              {availableLocales.map((l) => (
                <Menu.RadioItem key={l} value={l}>
                  {localeNames[l]}
                </Menu.RadioItem>
              ))}
            </Menu.RadioGroup>
          </Menu.Popup>
        </Menu.Root>
        <Link
          href={authed ? '/dashboard' : '/login'}
          className={buttonClass('outline', 'sm', 'ml-1')}
        >
          {authed ? t('openConsole') : t('signIn')}
        </Link>
      </div>
    </header>
  )
}
