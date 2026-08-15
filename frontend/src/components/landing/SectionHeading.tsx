'use client'

import { useTranslations } from 'next-intl'

// 各节的统一头部：§ 编号 + overline（mono 大写）+ Playfair 标题 + 副题 + 规则线。
// 印刷工序单式结构，替代卡片网格。
export function SectionHeading({
  index,
  section,
}: {
  index: string
  section: string
}) {
  const t = useTranslations(`landing.${section}`)

  return (
    <header className="max-w-3xl">
      <p className="font-code text-overline uppercase text-muted-foreground">
        §{index} · {t('overline')}
      </p>
      <h2 className="mt-3 font-display text-headline font-bold tracking-tight">
        {t('heading')}
      </h2>
      <p className="mt-3 text-body text-muted-foreground">{t('subheading')}</p>
      <div className="mt-8 h-px w-full bg-border" aria-hidden="true" />
    </header>
  )
}
