'use client'

import { useTranslations } from 'next-intl'
import type { DeliveryChannel, DeliveryHistoryItem } from '@/types/delivery'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

// B2.6 渠道健康：每渠道一行（状态点 + 名称 + 最近投递时间）。
// delivered → 正常(绿)；failed/expired → 异常(红)；无记录 → 空闲(灰)。
export function ChannelHealth({
  channels,
  latest,
  isLoading,
}: {
  channels: DeliveryChannel[]
  latest: DeliveryHistoryItem[]
  isLoading: boolean
}) {
  const t = useTranslations('dashboard.channelHealth')
  const { locale } = useLocaleContext()

  if (isLoading) {
    return (
      <ul className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <li key={i}>
            <Skeleton className="h-8 w-full" />
          </li>
        ))}
      </ul>
    )
  }

  if (channels.length === 0) {
    return <p className="text-sm text-muted-foreground">{t('noDelivery')}</p>
  }

  const latestByChannel = new Map<number, DeliveryHistoryItem>()
  for (const l of latest) {
    if (l.channel_id != null) latestByChannel.set(l.channel_id, l)
  }

  return (
    <ul className="space-y-2">
      {channels.map((ch) => {
        const row = latestByChannel.get(ch.id)
        const healthy =
          row != null &&
          (row.status === 'delivered' || row.status === 'pending')
        const abnormal = row != null && !healthy
        const idle = row == null
        return (
          <li
            key={ch.id}
            className="flex items-center justify-between gap-3 text-sm"
          >
            <span className="flex min-w-0 items-center gap-2">
              <span
                className={cn(
                  'size-2 shrink-0 rounded-full',
                  healthy && 'bg-success',
                  abnormal && 'bg-danger',
                  idle && 'bg-muted-foreground',
                )}
                aria-hidden="true"
              />
              <span className="truncate">{ch.name || '—'}</span>
            </span>
            <span
              className={cn(
                'shrink-0 text-xs',
                healthy && 'text-success',
                abnormal && 'text-danger',
                idle && 'text-muted-foreground',
              )}
            >
              {idle
                ? t('noDelivery')
                : row?.created_at
                  ? relativeTime(row.created_at, locale)
                  : ''}
            </span>
          </li>
        )
      })}
    </ul>
  )
}
