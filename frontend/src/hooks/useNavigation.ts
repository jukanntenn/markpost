'use client'

import {
  FileTextIcon,
  HistoryIcon,
  KeyRoundIcon,
  LayoutDashboardIcon,
  ScrollTextIcon,
  SendIcon,
  SettingsIcon,
  UsersIcon,
} from 'lucide-react'
import { useAuthReady } from '@/hooks/useAuthReady'

export interface NavItem {
  href: string
  labelKey: string
  icon: React.ComponentType<{ className?: string }>
}

export interface NavSection {
  headingKey?: string
  items: NavItem[]
}

// A2.6/A2.7 导航数据共享、渲染分离：桌面侧栏与移动 Sheet 消费同一份导航树，
// 角色化（admin 额外渲染"管理"分组）。active 判断见 K.3。
export function isNavActive(href: string, pathname: string): boolean {
  if (
    href === '/dashboard' ||
    href === '/posts' ||
    href === '/post-key' ||
    href === '/settings'
  ) {
    return pathname === href
  }
  return pathname === href || pathname.startsWith(`${href}/`)
}

export function useNavigation(): NavSection[] {
  const { isAdmin } = useAuthReady()

  const main: NavSection[] = [
    {
      items: [
        {
          href: '/dashboard',
          labelKey: 'navigation.dashboard',
          icon: LayoutDashboardIcon,
        },
        { href: '/posts', labelKey: 'navigation.posts', icon: FileTextIcon },
      ],
    },
    {
      headingKey: 'navigation.delivery',
      items: [
        {
          href: '/delivery/channels',
          labelKey: 'navigation.channels',
          icon: SendIcon,
        },
        {
          href: '/delivery/history',
          labelKey: 'navigation.history',
          icon: HistoryIcon,
        },
      ],
    },
    {
      items: [
        {
          href: '/post-key',
          labelKey: 'navigation.postKey',
          icon: KeyRoundIcon,
        },
        {
          href: '/settings',
          labelKey: 'navigation.settings',
          icon: SettingsIcon,
        },
      ],
    },
  ]

  if (!isAdmin) return main

  return [
    ...main,
    {
      headingKey: 'navigation.adminGroup',
      items: [
        {
          href: '/admin/dashboard',
          labelKey: 'navigation.dashboard',
          icon: LayoutDashboardIcon,
        },
        { href: '/admin/users', labelKey: 'navigation.users', icon: UsersIcon },
        {
          href: '/admin/posts',
          labelKey: 'navigation.posts',
          icon: FileTextIcon,
        },
        {
          href: '/admin/delivery/channels',
          labelKey: 'navigation.channels',
          icon: SendIcon,
        },
        {
          href: '/admin/delivery/history',
          labelKey: 'navigation.adminHistory',
          icon: HistoryIcon,
        },
        {
          href: '/admin/audit-logs',
          labelKey: 'navigation.auditLogs',
          icon: ScrollTextIcon,
        },
      ],
    },
  ]
}
