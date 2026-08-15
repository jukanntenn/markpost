'use client'

import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { ArrowDownIcon, ArrowRightIcon, ArrowUpRightIcon } from 'lucide-react'
import { useAuthReady } from '@/hooks/useAuthReady'
import { buttonClass } from '@/components/ui/button'
import { LANDING_CONTAINER, REPO_URL } from './links'
import { PostPageArtifact } from './PostPageArtifact'

// §01 Hero 对开页：标题自证源码（可见的「#」用 muted 灰 Fira Code，
// 其余是 Playfair 渲染结果），下方并置两件物证——固定不变的收件请求，
// 与每篇新生的公开页面。
export function HeroSection() {
  const t = useTranslations('landing')
  const tNav = useTranslations('landing.nav')
  const ts = useTranslations('landing.sample')
  const { hasHydrated, isAuthenticated } = useAuthReady()
  const authed = hasHydrated && isAuthenticated

  const request = [
    'POST /mpk-••••••••',
    '{',
    `  "title": "${ts('postTitle')}",`,
    `  ${t('hero.requestBody', { heading: ts('sectionHeading') })}`,
    '}',
  ].join('\n')

  return (
    <section className="pt-14 pb-16 md:pt-24 md:pb-24">
      <div className={LANDING_CONTAINER}>
        <p className="font-code text-overline uppercase text-muted-foreground">
          {t('hero.kicker')}
        </p>
        <h1 className="mt-5 max-w-4xl font-display text-display leading-(--text-display--line-height) font-bold tracking-tight">
          <span
            aria-hidden="true"
            className="mr-4 align-baseline font-code text-[0.4em] font-normal text-muted-foreground"
          >
            #
          </span>
          {t('hero.title1')}
          <br />
          {t('hero.title2')}
        </h1>
        <p className="mt-6 max-w-2xl text-subhead text-muted-foreground">
          {t('hero.subtitle')}
        </p>

        <div className="mt-10 flex flex-wrap items-center gap-x-6 gap-y-4">
          <Link
            href={authed ? '/dashboard' : '/login'}
            className={buttonClass('default', 'lg')}
          >
            {authed ? tNav('openConsole') : t('hero.getStarted')}
          </Link>
          <a
            href={REPO_URL}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 text-sm font-semibold text-foreground underline-offset-4 hover:underline"
          >
            {t('hero.viewGithub')}
            <ArrowUpRightIcon className="size-4" aria-hidden="true" />
          </a>
        </div>

        <div className="mt-14 grid items-center gap-8 md:mt-20 md:grid-cols-[1fr_auto_1fr] lg:gap-12">
          <figure className="min-w-0">
            <figcaption className="mb-3 font-code text-caption text-muted-foreground">
              {t('hero.inputCaption')}
            </figcaption>
            <div className="rounded-lg border border-border bg-surface p-5">
              <pre className="overflow-x-auto font-code text-small leading-relaxed text-foreground">
                {request}
              </pre>
              <p className="mt-4 border-t border-border pt-4 font-code text-small text-success">
                201 Created · 84ms
              </p>
            </div>
          </figure>

          <div className="flex items-center justify-center" aria-hidden="true">
            <ArrowRightIcon className="hidden size-6 text-muted-foreground md:block" />
            <ArrowDownIcon className="size-6 text-muted-foreground md:hidden" />
          </div>

          <figure className="min-w-0">
            <figcaption className="mb-3 font-code text-caption text-muted-foreground">
              {t('hero.outputCaption')}
            </figcaption>
            <PostPageArtifact />
          </figure>
        </div>
      </div>
    </section>
  )
}
