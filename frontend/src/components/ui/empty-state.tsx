import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

// B3.1/H.2 统一空态：图标 + 标题 + 说明 + 可选 CTA。
// 可操作空态（渠道/用户：有创建按钮）与信息空态（帖子：需说明）同构。
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed px-6 py-16 text-center',
        className,
      )}
    >
      {Icon && (
        <Icon className="size-10 text-muted-foreground" aria-hidden="true" />
      )}
      <h3 className="font-display text-subhead font-bold">{title}</h3>
      {description && (
        <p className="max-w-md text-sm text-muted-foreground">{description}</p>
      )}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}
