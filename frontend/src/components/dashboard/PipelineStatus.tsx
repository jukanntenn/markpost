'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import type { TodayCounts } from '@/types/delivery'

// B2.4 管道状态机（5 态）。优先级：全部失败 > 部分异常 > 空闲 > 运行中；
// 未配置覆盖一切（K.7 B2-2）。点击今日计数 → 投递历史页筛选。
export type PipelineState =
  'running' | 'partialFailure' | 'allFailed' | 'idle' | 'unconfigured'

const stateStyle: Record<
  PipelineState,
  { dot: string; pulse: boolean; text: string }
> = {
  running: { dot: 'bg-success', pulse: true, text: 'text-success' },
  partialFailure: { dot: 'bg-warning', pulse: false, text: 'text-warning' },
  allFailed: { dot: 'bg-danger', pulse: false, text: 'text-danger' },
  idle: {
    dot: 'bg-muted-foreground',
    pulse: false,
    text: 'text-muted-foreground',
  },
  unconfigured: {
    dot: 'bg-muted-foreground',
    pulse: false,
    text: 'text-muted-foreground',
  },
}

export function PipelineStatus({
  state,
  today,
  onTodayClick,
}: {
  state: PipelineState
  today?: TodayCounts
  onTodayClick: () => void
}) {
  const t = useTranslations('dashboard.pipeline')
  const tRoot = useTranslations()

  const style = stateStyle[state]

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card px-4 py-3">
      <div className="flex items-center gap-2.5">
        <span className="relative flex size-2.5">
          {style.pulse && (
            <span
              className={cn(
                'absolute inline-flex h-full w-full animate-ping rounded-full opacity-60',
                style.dot,
              )}
            />
          )}
          <span
            className={cn(
              'relative inline-flex size-2.5 rounded-full',
              style.dot,
            )}
          />
        </span>
        <span className={cn('text-sm font-semibold', style.text)}>
          {t(state)}
        </span>
      </div>

      {today && (
        <button
          type="button"
          onClick={onTodayClick}
          className="rounded-md px-2 py-1 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
        >
          {tRoot('dashboard.todayDelivery', {
            total: today.delivered + today.failed,
            success: today.delivered,
            failed: today.failed,
          })}
        </button>
      )}
    </div>
  )
}
