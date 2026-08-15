'use client'

import { useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { LandingHeader } from './LandingHeader'
import { HeroSection } from './HeroSection'
import { WorkflowSection } from './WorkflowSection'
import { PageSection } from './PageSection'
import { DeliverySection } from './DeliverySection'
import { OpenSourceSection } from './OpenSourceSection'
import { ColophonFooter } from './ColophonFooter'

// 着陆页：六个元素（Masthead / Hero / 原理 / 产物 / 投递 / 开源）+
// Colophon 页脚。每节 = 一个主张 + 一件物证 + 一行可验证的事实，
// 详见 specs/frontend/routes.md 与 landing 各节组件。
export function LandingPage() {
  const t = useTranslations('landing')
  const tNav = useTranslations('navigation')

  useEffect(() => {
    document.title = t('meta.title')
  }, [t])

  return (
    <div className="min-h-svh">
      <a href="#main-content" className="skip-link">
        {tNav('aria.skipToContent')}
      </a>
      <LandingHeader />
      <main id="main-content">
        <HeroSection />
        <div className="space-y-24 md:space-y-32">
          <WorkflowSection />
          <PageSection />
          <DeliverySection />
          <OpenSourceSection />
        </div>
      </main>
      <ColophonFooter />
    </div>
  )
}
