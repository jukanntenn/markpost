'use client'

import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { ErrorState } from '@/components/ui/error-state'
import { Button } from '@/components/ui/button'

// A2.8 层级1：渲染错误恢复。reset() 重试保住 TanStack Query 缓存与表单草稿。
export default function Error({
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  const t = useTranslations('error')
  const router = useRouter()

  return (
    <ErrorState title={t('title')} description={t('description')}>
      <Button onClick={() => reset()}>{t('retry')}</Button>
      <Button variant="outline" onClick={() => router.push('/dashboard')}>
        {t('backHome')}
      </Button>
    </ErrorState>
  )
}
