'use client'

import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { ShieldXIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'

// Q9 裁决：已登录非 admin 访问 /admin/* → 友好"无权限"状态页
// （文案六原则：不指责、可操作、给恢复路径）。
export function ForbiddenState() {
  const t = useTranslations('forbidden')
  const router = useRouter()

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
      <ShieldXIcon
        className="size-12 text-muted-foreground"
        aria-hidden="true"
      />
      <h1 className="font-display text-section font-bold">{t('title')}</h1>
      <p className="max-w-md text-sm text-muted-foreground">
        {t('description')}
      </p>
      <Button variant="outline" onClick={() => router.push('/dashboard')}>
        {t('backToDashboard')}
      </Button>
    </div>
  )
}
