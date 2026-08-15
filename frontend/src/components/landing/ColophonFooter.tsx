'use client'

import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useAuthReady } from '@/hooks/useAuthReady'
import { buttonClass } from '@/components/ui/button'
import { LANDING_CONTAINER, REPO_URL, DOCS_URL, DOCKER_HUB_URL } from './links'

// §06 Colophon：版权页式页脚——收尾 CTA 之后，是一段本页排印说明，
// 像一本书的版权页那样为页面签名。
export function ColophonFooter() {
  const t = useTranslations('landing')
  const tNav = useTranslations('landing.nav')
  const { hasHydrated, isAuthenticated } = useAuthReady()
  const authed = hasHydrated && isAuthenticated

  return (
    <footer className="border-t border-border">
      <div className={`py-16 md:py-20 ${LANDING_CONTAINER}`}>
        <div className="mx-auto max-w-xl text-center">
          <h2 className="font-display text-headline font-bold tracking-tight">
            {t('colophon.heading')}
          </h2>
          <p className="mt-3 text-body text-muted-foreground">
            {t('colophon.subheading')}
          </p>
          <div className="mt-8">
            <Link
              href={authed ? '/dashboard' : '/login'}
              className={buttonClass('default', 'lg')}
            >
              {authed ? tNav('openConsole') : t('colophon.getStarted')}
            </Link>
          </div>
        </div>

        <nav
          aria-label="markpost"
          className="mt-16 flex flex-wrap items-center justify-center gap-x-8 gap-y-3 text-small"
        >
          <a
            href={DOCS_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            {t('colophon.docs')}
          </a>
          <a
            href={REPO_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            GitHub
          </a>
          <a
            href={DOCKER_HUB_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            {t('colophon.dockerHub')}
          </a>
        </nav>

        <div className="mt-6 flex flex-col gap-3 border-t border-border pt-6 md:flex-row md:items-center md:justify-between">
          <p className="text-caption text-muted-foreground">
            {t('colophon.copyright', { year: new Date().getFullYear() })}
          </p>
          <p className="text-caption text-muted-foreground">
            {t('colophon.typeset')}
          </p>
        </div>
      </div>
    </footer>
  )
}
