import { cn } from '@/lib/utils'

// UserAvatar：GitHub 用户显示其 GitHub 头像；密码注册用户（无 avatar_url）
// 回退为用户名首字母圆。
export function UserAvatar({
  username,
  avatarUrl,
  className,
}: {
  username: string
  avatarUrl?: string | null
  className?: string
}) {
  if (avatarUrl) {
    return (
      // eslint-disable-next-line @next/next/no-img-element -- 远程头像：静态导出没有图片优化器
      <img
        src={avatarUrl}
        alt=""
        className={cn(
          'size-6 shrink-0 rounded-full bg-muted object-cover',
          className,
        )}
      />
    )
  }
  return (
    <span
      aria-hidden="true"
      className={cn(
        'flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold text-muted-foreground uppercase select-none',
        className,
      )}
    >
      {username.charAt(0)}
    </span>
  )
}
