'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useAuthReady } from '@/hooks/useAuthReady'
import { buttonClass } from '@/components/ui/button'
import {
  APP_VERSION,
  DOCS_URL,
  DOCKER_HUB_URL,
  ISSUES_URL,
  LICENSE_URL,
  REPO_URL,
} from '@/lib/site'
import { LANDING_CONTAINER } from './links'

// §06 收尾：结束 CTA 之下是站点 footer——品牌区 + 资源/项目两列 + 底条
// （版权 · 许可 · 版本 · 排印说明，像一本书的版权页那样为页面签名）。
export function SiteFooter() {
  const t = useTranslations('landing')
  const tNav = useTranslations('landing.nav')
  const { hasHydrated, isAuthenticated } = useAuthReady()
  const authed = hasHydrated && isAuthenticated

  const linkClass =
    'text-small text-muted-foreground underline-offset-4 hover:text-foreground hover:underline'
  const headingClass = 'text-small font-semibold text-foreground'

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

        <div className="mt-16 flex flex-col gap-10 border-t border-border pt-10 md:grid md:grid-cols-[minmax(0,1fr)_auto_auto] md:gap-16">
          <div>
            <span className="inline-flex items-center gap-2">
              <Image
                src="/markpost.svg"
                alt=""
                className="h-5 w-auto"
                width={24}
                height={24}
              />
              <span className="font-display text-body font-bold tracking-tight">
                markpost
              </span>
            </span>
            <p className="mt-3 max-w-xs text-small text-muted-foreground">
              {t('colophon.tagline')}
            </p>
          </div>

          <div className="grid grid-cols-2 gap-8 md:contents">
            <nav aria-label={t('colophon.resources')}>
              <h3 className={headingClass}>{t('colophon.resources')}</h3>
              <ul className="mt-3 space-y-2.5">
                <li>
                  <a
                    href={DOCS_URL}
                    target="_blank"
                    rel="noreferrer"
                    className={linkClass}
                  >
                    {t('colophon.docs')}
                  </a>
                </li>
                <li>
                  <a
                    href={ISSUES_URL}
                    target="_blank"
                    rel="noreferrer"
                    className={linkClass}
                  >
                    {t('colophon.issues')}
                  </a>
                </li>
              </ul>
            </nav>
            <nav aria-label={t('colophon.project')}>
              <h3 className={headingClass}>{t('colophon.project')}</h3>
              <ul className="mt-3 space-y-2.5">
                <li>
                  <a
                    href={REPO_URL}
                    target="_blank"
                    rel="noreferrer"
                    className={linkClass}
                  >
                    GitHub
                  </a>
                </li>
                <li>
                  <a
                    href={DOCKER_HUB_URL}
                    target="_blank"
                    rel="noreferrer"
                    className={linkClass}
                  >
                    {t('colophon.dockerHub')}
                  </a>
                </li>
                <li>
                  <a
                    href={LICENSE_URL}
                    target="_blank"
                    rel="noreferrer"
                    className={linkClass}
                  >
                    MIT License
                  </a>
                </li>
              </ul>
            </nav>
          </div>
        </div>

        <div className="mt-10 flex flex-col gap-3 border-t border-border pt-6 md:flex-row md:items-center md:justify-between">
          <p className="text-caption text-muted-foreground">
            {t('colophon.copyright', { year: new Date().getFullYear() })} · v
            {APP_VERSION}
          </p>
          <p className="text-caption text-muted-foreground">
            {t('colophon.typeset')}
          </p>
        </div>
      </div>
    </footer>
  )
}
