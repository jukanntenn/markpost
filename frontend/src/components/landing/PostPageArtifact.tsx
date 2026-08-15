'use client'

import { useTranslations } from 'next-intl'

// 复刻 backend/templates/post.html 的公开文章页：它是一张独立的冷色 slate
// 纸面（系统字体、#0f172a 墨色），与 Ember Studio 暖色 landing 刻意保持
// 材质差异——物证看起来应当像从产品里揭下来的一页，而不是页面装饰。
export function PostPageArtifact() {
  const t = useTranslations('landing.sample')

  return (
    <div className="rounded-lg bg-[#f6f7f9] p-4 font-sans sm:p-6 dark:bg-[#0b1220]">
      <article className="rounded-[16px] border border-[#e5e7eb] bg-white p-5 shadow-[0_10px_30px_rgba(15,23,42,0.08)] sm:p-8 dark:border-[#94a3b838] dark:bg-[#0f172a] dark:shadow-none">
        <header className="border-b border-[#e5e7eb] pb-4 dark:border-[#94a3b838]">
          <h3 className="text-xl leading-[1.2] font-semibold tracking-[-0.02em] text-[#0f172a] sm:text-2xl dark:text-[#e5e7eb]">
            {t('postTitle')}
          </h3>
          <p className="mt-1.5 text-xs text-[#475569] dark:text-[#a1a1aa]">
            {t('postTime')} · /p-Vq2mXk
          </p>
        </header>
        <div className="mt-5 space-y-3 text-sm leading-[1.85]">
          <p className="text-base font-semibold text-[#0f172a] dark:text-[#e5e7eb]">
            {t('sectionHeading')}
          </p>
          <p className="text-[#334155] dark:text-[#cbd5e1]">{t('paragraph')}</p>
          <ul className="space-y-1.5 pl-4 text-[#334155] dark:text-[#cbd5e1]">
            <li className="list-disc marker:text-[#94a3b8]">{t('bullet1')}</li>
            <li className="list-disc marker:text-[#94a3b8]">{t('bullet2')}</li>
          </ul>
        </div>
        <footer className="mt-6 border-t border-[#e5e7eb] pt-3 text-center text-xs text-[#475569]/60 dark:border-[#94a3b838] dark:text-[#a1a1aa]/60">
          Powered by Markpost
        </footer>
      </article>
    </div>
  )
}
