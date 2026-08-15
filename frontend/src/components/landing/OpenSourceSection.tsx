'use client'

import { useTranslations } from 'next-intl'
import { ArrowUpRightIcon } from 'lucide-react'
import { LANDING_CONTAINER, DOCS_URL } from './links'
import { SectionHeading } from './SectionHeading'

// §05 开源：信任背书而非部署教程——三行 compose 是物证，规格表只列
// 可验证的事实。传递的立场是「随时可以搬走，所以可以放心留下来」。
export function OpenSourceSection() {
  const t = useTranslations('landing.oss')

  const specs = [
    { key: t('license'), value: t('licenseValue') },
    { key: t('stack'), value: t('stackValue') },
    { key: t('retention'), value: t('retentionValue') },
    { key: t('rateLimit'), value: t('rateLimitValue') },
  ]

  return (
    <section id="open-source">
      <div className={LANDING_CONTAINER}>
        <SectionHeading index="05" section="oss" />
        <div className="mt-10 grid gap-12 lg:grid-cols-2 lg:gap-16">
          <div>
            <p className="mb-3 font-code text-caption text-muted-foreground">
              {t('composeCaption')}
            </p>
            <pre className="overflow-x-auto rounded-lg border border-border bg-surface p-5 font-code text-small leading-relaxed text-foreground">
              {`services:
  markpost:
    image: jukanntenn/markpost
  postgres:
    image: postgres:17-alpine`}
            </pre>
            <a
              href={DOCS_URL}
              target="_blank"
              rel="noreferrer"
              className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-primary underline-offset-4 hover:underline"
            >
              {t('quickstart')}
              <ArrowUpRightIcon className="size-4" aria-hidden="true" />
            </a>
          </div>
          <dl className="divide-y divide-border border-y border-border">
            {specs.map((spec) => (
              <div
                key={spec.key}
                className="grid grid-cols-[7rem_1fr] gap-4 py-3.5 sm:grid-cols-[9rem_1fr]"
              >
                <dt className="text-small text-muted-foreground">{spec.key}</dt>
                <dd className="text-small font-semibold">{spec.value}</dd>
              </div>
            ))}
          </dl>
        </div>
      </div>
    </section>
  )
}
