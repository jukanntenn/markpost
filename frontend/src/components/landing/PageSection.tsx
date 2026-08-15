'use client'

import { useTranslations } from 'next-intl'
import { MoveRightIcon } from 'lucide-react'
import { LANDING_CONTAINER } from './links'
import { SectionHeading } from './SectionHeading'
import { PostPageArtifact } from './PostPageArtifact'

// §03 产物：页边注式批注版式——左列四条批注以细线牵引指向右侧的
// 文章页物证，替代功能卡片网格。
export function PageSection() {
  const t = useTranslations('landing.page')
  const tHero = useTranslations('landing.hero')

  const notes = [
    { kicker: t('note1.kicker'), text: t('note1.text') },
    { kicker: t('note2.kicker'), text: t('note2.text') },
    { kicker: t('note3.kicker'), text: t('note3.text') },
    { kicker: t('note4.kicker'), text: t('note4.text') },
  ]

  return (
    <section id="page">
      <div className={LANDING_CONTAINER}>
        <SectionHeading index="03" section="page" />
        <div className="mt-10 grid gap-10 lg:grid-cols-[5fr_7fr] lg:gap-16">
          <ol className="divide-y divide-border border-y border-border">
            {notes.map((note) => (
              <li key={note.kicker} className="flex items-center gap-6 py-5">
                <div className="flex-1">
                  <p className="text-small font-semibold">{note.kicker}</p>
                  <p className="mt-1 text-small text-muted-foreground">
                    {note.text}
                  </p>
                </div>
                <span
                  className="hidden items-center gap-2 lg:flex"
                  aria-hidden="true"
                >
                  <span className="h-px w-16 bg-border" />
                  <MoveRightIcon className="size-4 shrink-0 text-muted-foreground" />
                </span>
              </li>
            ))}
          </ol>
          <figure className="min-w-0">
            <figcaption className="mb-3 font-code text-caption text-muted-foreground">
              {tHero('outputCaption')}
            </figcaption>
            <PostPageArtifact />
          </figure>
        </div>
      </div>
    </section>
  )
}
