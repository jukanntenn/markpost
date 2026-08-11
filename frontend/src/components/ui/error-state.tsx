import * as React from 'react'
import { CircleAlertIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

// A2.8 错误态 UI 共用组件：图标 + 标题 + 描述 + 主/次 CTA。
// 接收纯文案（global-error 场景无 i18n Provider，由调用方传入字符串）。
export function ErrorState({
  className,
  title,
  description,
  children,
}: {
  className?: string
  title: string
  description?: string
  children?: React.ReactNode
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-4 py-20 text-center',
        className,
      )}
    >
      <CircleAlertIcon
        className="size-12 text-muted-foreground"
        aria-hidden="true"
      />
      <h1 className="font-display text-section font-bold">{title}</h1>
      {description && (
        <p className="max-w-md text-sm text-muted-foreground">{description}</p>
      )}
      {children && (
        <div className="mt-2 flex items-center gap-2">{children}</div>
      )}
    </div>
  )
}
