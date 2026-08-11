'use client'

import { ApiError, ApiErrorCodes } from '@/lib/api'
import { useTranslations } from 'next-intl'
import { CircleAlertIcon, WifiOffIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'

// B3.1/H.2 统一三态范式：loading → 内容结构骨架（非 spinner）；
// error → 图标 + 说明 + [重试]（区分网络/服务端错误，I.8 恢复路径）；
// empty → 由调用方渲染 EmptyState；success → children。
export function ListState({
  isLoading,
  error,
  loadingSkeleton,
  empty,
  emptyWhen,
  onRetry,
  errorTitle,
  children,
}: {
  isLoading: boolean
  error: Error | null
  loadingSkeleton: React.ReactNode
  empty?: React.ReactNode
  emptyWhen?: boolean
  onRetry?: () => void
  // 覆盖默认错误标题（如 D2.2 "异常检测暂不可用"）。
  errorTitle?: string
  children: React.ReactNode
}) {
  const t = useTranslations('common')
  const tNetwork = useTranslations('network')
  const tPosts = useTranslations('posts')

  if (isLoading) return <>{loadingSkeleton}</>

  if (error) {
    const isNetwork =
      error instanceof ApiError &&
      (error.code === ApiErrorCodes.NetworkError ||
        error.code === ApiErrorCodes.Timeout)
    return (
      <div
        role="alert"
        className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed px-6 py-16 text-center"
      >
        {isNetwork ? (
          <WifiOffIcon
            className="size-10 text-muted-foreground"
            aria-hidden="true"
          />
        ) : (
          <CircleAlertIcon
            className="size-10 text-muted-foreground"
            aria-hidden="true"
          />
        )}
        <p className="max-w-md text-sm text-foreground">
          {isNetwork ? tNetwork('offline') : (errorTitle ?? tPosts('error'))}
        </p>
        {onRetry && (
          <Button variant="outline" onClick={onRetry}>
            {t('retry')}
          </Button>
        )}
      </div>
    )
  }

  if (emptyWhen) return <>{empty}</>

  return <>{children}</>
}
