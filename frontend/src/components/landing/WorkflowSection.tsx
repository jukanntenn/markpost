'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { LANDING_CONTAINER } from './links'
import { SectionHeading } from './SectionHeading'

// §02 原理：印刷工序单——三道工序横向排开，上下以规则线收束，
// 竖细线分隔；Playfair 大号数字 + 每站一枚 mono 小物证。
export function WorkflowSection() {
  const t = useTranslations('landing.workflow')

  const steps = [
    {
      n: '01',
      name: t('step1.name'),
      desc: t('step1.desc'),
      hint: t('step1.hint'),
    },
    {
      n: '02',
      name: t('step2.name'),
      desc: t('step2.desc'),
      hint: t('step2.hint'),
    },
    {
      n: '03',
      name: t('step3.name'),
      desc: t('step3.desc'),
      hint: t('step3.hint'),
    },
  ]

  return (
    <section id="how-it-works">
      <div className={LANDING_CONTAINER}>
        <SectionHeading index="02" section="workflow" />
        <ol className="mt-10 grid gap-10 md:grid-cols-3 md:gap-0">
          {steps.map((step, i) => (
            <li
              key={step.n}
              className={cn(
                'md:px-8',
                i === 0 ? 'md:pl-0' : 'md:border-l md:border-border',
                i === steps.length - 1 && 'md:pr-0',
              )}
            >
              <p className="font-display text-headline font-bold text-muted-foreground">
                {step.n}
              </p>
              <h3 className="mt-3 font-display text-section font-bold">
                {step.name}
              </h3>
              <p className="mt-2 text-small leading-relaxed text-muted-foreground">
                {step.desc}
              </p>
              <code className="mt-4 inline-block rounded-sm border border-border bg-surface px-2 py-1 font-code text-caption text-secondary-foreground">
                {step.hint}
              </code>
            </li>
          ))}
        </ol>
      </div>
    </section>
  )
}
