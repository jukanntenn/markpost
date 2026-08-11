'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import {
  ChevronDownIcon,
  ChevronRightIcon,
  CircleCheckIcon,
  CircleXIcon,
  Loader2Icon,
} from 'lucide-react'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { buttonClass } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { DeliveryHistoryItem } from '@/types/delivery'

// B2.5/K.2 活动事件单元：文章 + 投递结果原子。
// 全成功事件投递子项默认折叠（只显示摘要），点击摘要行展开；
// 部分失败事件默认展开失败项（成功项可折叠）。
export interface ActivityEvent {
  postQid: string
  postTitle: string
  createdAt: string
  deliveries: DeliveryHistoryItem[]
  pendingCount: number
}

export function EventUnit({ event }: { event: ActivityEvent }) {
  const t = useTranslations('dashboard.activity')
  const { locale } = useLocaleContext()

  const total = event.deliveries.length + event.pendingCount
  const done = event.deliveries.filter((d) => d.status === 'delivered').length
  const hasFailure = event.deliveries.some(
    (d) => d.status === 'failed' || d.status === 'expired',
  )
  const allPending = event.pendingCount > 0 && event.deliveries.length === 0
  // 有失败 → 默认展开（用户要看原因）；全成功 → 默认折叠。
  const [expanded, setExpanded] = useState(hasFailure)

  const summary = allPending
    ? t('pending')
    : hasFailure
      ? t('partialFail', { done, total })
      : t('delivered', { done, total })

  const toggleLabel = expanded ? t('hideDetails') : t('showDetails')

  return (
    <article className="rounded-lg border bg-card p-4 transition-shadow duration-150 hover:shadow-card-hover">
      <div className="flex items-start justify-between gap-3">
        <h3 className="min-w-0 font-display text-subhead font-bold">
          <a
            href={`/${event.postQid}`}
            target="_blank"
            rel="noreferrer"
            title={event.postTitle}
            className="block truncate underline-offset-4 hover:underline"
          >
            {event.postTitle || event.postQid}
          </a>
        </h3>
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          aria-label={toggleLabel}
          className="flex shrink-0 items-center gap-1.5 rounded-md px-1 py-0.5 text-sm font-medium outline-none focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
        >
          <span
            className={cn(
              allPending
                ? 'text-muted-foreground'
                : hasFailure
                  ? 'text-warning'
                  : 'text-success',
            )}
          >
            {summary}
          </span>
          {expanded ? (
            <ChevronDownIcon className="size-4 text-muted-foreground" />
          ) : (
            <ChevronRightIcon className="size-4 text-muted-foreground" />
          )}
        </button>
      </div>
      <p className="mt-0.5 text-xs text-muted-foreground">
        {relativeTime(event.createdAt, locale)}
      </p>

      {expanded && (
        <ul className="mt-3 space-y-1">
          {event.deliveries.map((d) => {
            const failed = d.status === 'failed' || d.status === 'expired'
            return (
              <li
                key={d.id}
                className={cn(
                  'flex items-center justify-between gap-3 rounded-md px-2 py-1.5 text-sm',
                  failed && 'bg-danger/5',
                )}
              >
                <span className="flex min-w-0 items-center gap-2">
                  {failed ? (
                    <CircleXIcon className="size-4 shrink-0 text-danger" />
                  ) : (
                    <CircleCheckIcon className="size-4 shrink-0 text-success" />
                  )}
                  <span className="truncate">{d.channel_name ?? '—'}</span>
                </span>
                {failed ? (
                  <span className="flex shrink-0 items-center gap-2">
                    <span
                      className="max-w-44 truncate text-xs text-danger"
                      title={d.last_error}
                    >
                      {d.last_error || '—'}
                    </span>
                    <Link
                      href="/delivery/history"
                      className={buttonClass('link', 'xs')}
                    >
                      {t('viewDetail')}
                    </Link>
                  </span>
                ) : (
                  <span className="shrink-0 text-xs text-success">✓</span>
                )}
              </li>
            )
          })}
          {Array.from({ length: event.pendingCount }).map((_, i) => (
            <li
              key={`pending-${i}`}
              className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground"
            >
              <Loader2Icon className="size-4 shrink-0 animate-spin" />
              <span className="truncate">{t('pending')}</span>
            </li>
          ))}
        </ul>
      )}
    </article>
  )
}
