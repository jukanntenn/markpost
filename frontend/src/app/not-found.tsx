'use client'

import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useAuthReady } from '@/hooks/useAuthReady'
import { ErrorState } from '@/components/ui/error-state'
import { buttonClass } from '@/components/ui/button'

// A2.8 层级1：品牌化 404。已登录 → 回 Dashboard；未登录 → 前往登录。
// （已登录无权访问 /admin 的友好提示由 AdminRoute/ForbiddenState 处理。）
export default function NotFound() {
  const t = useTranslations('notFound')
  const { hasHydrated, isAuthenticated } = useAuthReady()
  const authed = hasHydrated && isAuthenticated

  return (
    <ErrorState title={t('title')} description={t('pageNotFound')}>
      {authed ? (
        <Link href="/dashboard" className={buttonClass('default')}>
          {t('backToDashboard')}
        </Link>
      ) : (
        <Link href="/login" className={buttonClass('default')}>
          {t('goToLogin')}
        </Link>
      )}
    </ErrorState>
  )
}
