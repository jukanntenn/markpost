'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { isNavActive, useNavigation, type NavItem } from '@/hooks/useNavigation'
import { cn } from '@/lib/utils'

// A2.6 数据共享、渲染分离：桌面与移动共用同一份导航树，仅视觉/交互分离。
// 桌面 item 高 h-9；移动 item 高 h-11（44px，WCAG 2.5.5 触控目标）。
export function SidebarNav({
  onNavigate,
  itemClassName,
}: {
  onNavigate?: () => void
  itemClassName?: string
}) {
  const t = useTranslations()
  const pathname = usePathname()
  const sections = useNavigation()

  return (
    <nav aria-label={t('navigation.aria.sidebar')}>
      {sections.map((section, i) => (
        <div key={i} className="mb-6">
          {section.headingKey && (
            <p className="mb-1.5 px-3 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
              {t(section.headingKey)}
            </p>
          )}
          <ul className="flex flex-col gap-0.5">
            {section.items.map((item) => (
              <SidebarLink
                key={item.href}
                item={item}
                active={isNavActive(item.href, pathname)}
                onClick={onNavigate}
                className={itemClassName}
              />
            ))}
          </ul>
        </div>
      ))}
    </nav>
  )
}

function SidebarLink({
  item,
  active,
  onClick,
  className,
}: {
  item: NavItem
  active: boolean
  onClick?: () => void
  className?: string
}) {
  const t = useTranslations()
  const Icon = item.icon
  return (
    <li>
      <Link
        href={item.href}
        onClick={onClick}
        data-active={active || undefined}
        aria-current={active ? 'page' : undefined}
        className={cn(
          'flex items-center gap-3 rounded-md border-l-2 border-transparent px-3 text-sm font-medium text-muted-foreground transition-colors duration-150 hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring data-[active]:border-primary data-[active]:bg-accent data-[active]:text-foreground',
          'h-9', // 桌面
          className,
        )}
      >
        <Icon className="size-4 shrink-0" />
        <span className="truncate">{t(item.labelKey)}</span>
      </Link>
    </li>
  )
}
