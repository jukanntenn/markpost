'use client'

import { useEffect, useRef, useState } from 'react'
import Link from 'next/link'
import Image from 'next/image'
import { usePathname, useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import {
  ChevronDownIcon,
  LogOutIcon,
  MenuIcon,
  SettingsIcon,
  ShieldIcon,
  UserIcon,
} from 'lucide-react'
import { useAuthReady } from '@/hooks/useAuthReady'
import { useProfileSync } from '@/hooks/useProfileSync'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/lib/api'
import { toastManager } from '@/stores/toast'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Button } from '@/components/ui/button'
import { Menu } from '@/components/ui/menu'
import { VipBadge } from '@/components/ui/vip-badge'
import { SidebarNav } from '@/components/layout/SidebarNav'
import { MobileSidebar } from '@/components/layout/MobileSidebar'
import { APP_VERSION, DOCS_URL, REPO_URL } from '@/lib/site'

// A2.5 统一应用壳：顶栏退化为全局工具栏（无主导航），所有用户走侧栏，
// 侧栏内容随角色变化。桌面 sticky 侧栏 + 移动 Dialog Sheet 消费同一导航树。
export function AppShell({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const t = useTranslations('navigation')
  const tCommon = useTranslations('common')
  const tNetwork = useTranslations('network')
  const tFooter = useTranslations('footer')
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const { isAuthenticated, isAdmin } = useAuthReady()
  // 管理端改动（vip/封禁等）落库后，登录快照不会自己更新：每次整页加载
  // 拉一次 /me 覆盖本地快照。
  useProfileSync()

  const [mobileOpen, setMobileOpen] = useState(false)
  const [scrolled, setScrolled] = useState(false)
  const announcedRef = useRef('')

  // 路由切换自动关闭移动 Sheet（A2.2 交互完整）
  useEffect(() => {
    setMobileOpen(false)
  }, [pathname])

  // 顶栏滚动加下边框（design.md：nav 滚动时 1px bottom border）
  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 0)
    window.addEventListener('scroll', handleScroll, { passive: true })
    handleScroll()
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  // J.5 路由切换 announce（aria-live 区域）
  useEffect(() => {
    const title = document.title
    if (title && announcedRef.current !== title) {
      announcedRef.current = title
      document
        .getElementById('route-announcer')
        ?.replaceChildren(document.createTextNode(title))
    }
  }, [pathname])

  // B1 场景 E：后端 logout 失败也强制本地登出，但提示部分会话可能残留
  const handleLogout = async () => {
    try {
      await authApi.logout()
    } catch {
      toastManager.add({ type: 'warning', title: tNetwork('logoutPartial') })
    } finally {
      logout()
      router.replace('/login')
    }
  }

  return (
    <div className="flex min-h-svh flex-col">
      <a href="#main-content" className="skip-link">
        {t('aria.skipToContent')}
      </a>
      <p id="route-announcer" className="sr-only" aria-live="polite" />

      <header
        className={`sticky top-0 z-50 h-(--header-height) w-full bg-background/80 backdrop-blur transition-[border-color] duration-150 ${scrolled ? 'border-b' : ''}`}
      >
        <div className="mx-auto flex h-full max-w-(--container-max) items-center gap-2 px-4 md:px-6 lg:px-8">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t('aria.openMenu')}
            className="size-11 lg:hidden"
            onClick={() => setMobileOpen(true)}
          >
            <MenuIcon className="size-5" />
          </Button>
          <Link
            href="/dashboard"
            aria-label="markpost"
            className="flex h-11 items-center rounded-md px-1 transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
          >
            <Image
              src="/markpost.svg"
              alt=""
              className="h-6 w-auto"
              width={24}
              height={24}
            />
          </Link>
          <div className="flex-1" />
          <ThemeToggle />
          {isAuthenticated && (
            <Menu.Root>
              <Menu.Trigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    className="h-11 gap-2 lg:h-10"
                  />
                }
              >
                <UserIcon className="size-4" />
                <span className="hidden sm:inline">
                  {user?.username || tCommon('user')}
                </span>
                {user?.vip && <VipBadge />}
                <ChevronDownIcon className="size-4 text-muted-foreground" />
              </Menu.Trigger>
              <Menu.Popup>
                <Menu.Group>
                  <Menu.Label className="flex items-center gap-1.5">
                    {user?.username || tCommon('user')}
                    {user?.vip && <VipBadge />}
                  </Menu.Label>
                </Menu.Group>
                <Menu.Separator />
                {isAdmin && (
                  <Menu.Item onClick={() => router.push('/admin/dashboard')}>
                    <ShieldIcon className="size-4" />
                    {t('userMenu.admin')}
                  </Menu.Item>
                )}
                <Menu.Item onClick={() => router.push('/settings')}>
                  <SettingsIcon className="size-4" />
                  {t('userMenu.settings')}
                </Menu.Item>
                <Menu.Separator />
                <Menu.Item variant="danger" onClick={handleLogout}>
                  <LogOutIcon className="size-4" />
                  {t('userMenu.logout')}
                </Menu.Item>
              </Menu.Popup>
            </Menu.Root>
          )}
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-(--container-max) flex-1">
        <aside className="sticky top-(--header-height) hidden h-[calc(100svh-var(--header-height))] w-64 shrink-0 border-r border-sidebar-border lg:block">
          <div className="h-full overflow-y-auto px-3 py-6">
            <SidebarNav />
          </div>
        </aside>
        <main
          id="main-content"
          tabIndex={-1}
          className="min-w-0 flex-1 px-4 py-6 outline-none md:px-6 lg:px-8 lg:py-12"
        >
          {children}
        </main>
      </div>

      {/* 底部 slim footer：自托管实例的运行版本 + 文档/源码入口，
          容器与顶栏对齐（design.md 页面容器约束）。 */}
      <footer className="border-t border-border">
        <div className="mx-auto flex max-w-(--container-max) flex-wrap items-center justify-between gap-x-6 gap-y-1 px-4 py-4 text-caption text-muted-foreground md:px-6 lg:px-8">
          <span>markpost v{APP_VERSION}</span>
          <nav aria-label={tFooter('docs')} className="flex items-center gap-4">
            <a
              href={DOCS_URL}
              target="_blank"
              rel="noreferrer"
              className="underline-offset-4 hover:text-foreground hover:underline"
            >
              {tFooter('docs')}
            </a>
            <a
              href={REPO_URL}
              target="_blank"
              rel="noreferrer"
              className="underline-offset-4 hover:text-foreground hover:underline"
            >
              GitHub
            </a>
          </nav>
        </div>
      </footer>

      <MobileSidebar open={mobileOpen} onOpenChange={setMobileOpen} />
    </div>
  )
}
