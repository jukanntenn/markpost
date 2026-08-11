'use client'

import { useTranslations } from 'next-intl'

import { PageHeading } from '@/components/ui/page-heading'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { AppSettingsCard } from './AppSettingsCard'
import { PasswordChangeCard } from './PasswordChangeCard'
import { SessionsCard } from './SessionsCard'

// F.2 设置页（精简后）：偏好（语言）+ 安全（改密 + 我的会话在同一"安全"
// 卡片内，I.12/F.2 一致性裁决）。
export function SettingsPage() {
  const t = useTranslations('settings')

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeading>{t('title')}</PageHeading>

      <AppSettingsCard />

      <Card>
        <CardHeader>
          <CardTitle>{t('security')}</CardTitle>
          <CardDescription>{t('securityDescription')}</CardDescription>
        </CardHeader>
        <CardContent>
          <PasswordChangeCard embedded />
          <Separator className="my-6" />
          <SessionsCard embedded />
        </CardContent>
      </Card>
    </div>
  )
}

export default SettingsPage
