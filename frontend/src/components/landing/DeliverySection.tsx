'use client'

import { useTranslations } from 'next-intl'
import { ArrowUpRightIcon, CheckIcon, FileTextIcon } from 'lucide-react'
import { LANDING_CONTAINER } from './links'
import { SectionHeading } from './SectionHeading'

// §04 投递：两件物证——真·飞书互动卡片复刻，与关键词过滤的语法样张。
// 飞书卡片用产品自身的灰底白卡与品牌蓝，是描画出来的「别处的界面」，
// 不占用 Ember 色板。
export function DeliverySection() {
  const t = useTranslations('landing.delivery')
  const ts = useTranslations('landing.sample')

  const expressions = [
    { expr: t('expr1'), meaning: t('or') },
    { expr: t('expr2'), meaning: t('and') },
    { expr: t('expr3'), meaning: t('group') },
  ]

  return (
    <section id="delivery">
      <div className={LANDING_CONTAINER}>
        <SectionHeading index="04" section="delivery" />
        <div className="mt-10 grid gap-12 lg:grid-cols-2 lg:gap-16">
          <figure className="min-w-0">
            <figcaption className="mb-3 font-code text-caption text-muted-foreground">
              {t('cardCaption')}
            </figcaption>
            <div className="rounded-lg bg-[#eef0f2] p-5 sm:p-6 dark:bg-[#1e1f22]">
              <div className="max-w-sm rounded-xl border border-[#e0e1e3] bg-white p-4 dark:border-[#3a3b3d] dark:bg-[#2b2c2f]">
                <div className="flex items-start gap-3">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-[#3370ff]/10 text-[#3370ff] dark:text-[#4e83fd]">
                    <FileTextIcon className="size-4" aria-hidden="true" />
                  </span>
                  <div className="min-w-0">
                    <p className="text-small font-semibold text-[#1f2329] dark:text-[#f1f1f2]">
                      {ts('postTitle')}
                    </p>
                    <p className="mt-1 line-clamp-2 text-small leading-relaxed text-[#646a73] dark:text-[#8f959e]">
                      {ts('preview')}
                    </p>
                  </div>
                </div>
                <div className="mt-3 border-t border-[#e0e1e3] pt-3 text-center dark:border-[#3a3b3d]">
                  <span className="inline-flex items-center gap-1 text-small font-medium text-[#3370ff] dark:text-[#4e83fd]">
                    {t('openPost')}
                    <ArrowUpRightIcon className="size-3.5" aria-hidden="true" />
                  </span>
                </div>
              </div>
            </div>
            <ul className="mt-5 space-y-2">
              <li className="flex items-center gap-2 text-small text-muted-foreground">
                <CheckIcon
                  className="size-4 shrink-0 text-success"
                  aria-hidden="true"
                />
                {t('reliability1')}
              </li>
              <li className="flex items-center gap-2 text-small text-muted-foreground">
                <CheckIcon
                  className="size-4 shrink-0 text-success"
                  aria-hidden="true"
                />
                {t('reliability2')}
              </li>
            </ul>
          </figure>

          <div>
            <h3 className="font-display text-section font-bold">
              {t('grammarHeading')}
            </h3>
            <dl className="mt-4 divide-y divide-border border-y border-border">
              {expressions.map((row) => (
                <div
                  key={row.expr}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-baseline gap-4 py-3.5"
                >
                  <dt className="truncate font-code text-small text-foreground">
                    {row.expr}
                  </dt>
                  <dd className="text-small text-muted-foreground">
                    {row.meaning}
                  </dd>
                </div>
              ))}
              <div className="grid grid-cols-[minmax(0,1fr)_auto] items-baseline gap-4 py-3.5">
                <dt className="font-code text-small text-muted-foreground">
                  —
                </dt>
                <dd className="text-small text-muted-foreground">
                  {t('empty')}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </div>
    </section>
  )
}
