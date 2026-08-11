'use client'

import { useAuthGuard } from '@/hooks/useAuthGuard'
import { AppShellSkeleton } from '@/components/providers/AppShellSkeleton'
import { ForbiddenState } from '@/components/auth/ForbiddenState'

interface AuthGateProps {
  shouldShow: (isAuthenticated: boolean, isAdmin: boolean) => boolean
  showSpinnerWhen?: (isAuthenticated: boolean, isAdmin: boolean) => boolean
  redirectPath: string
  // K.3 intended-URL：redirect 时携带 ?next=
  withNext?: boolean
  // 已登录但无权限时渲染"无权限"友好态而非重定向（admin 路由）
  showForbidden?: boolean
  children: React.ReactNode
}

// A2.9：auth hydrate 期间渲染统一 AppShellSkeleton（替代旧 PageSpinner）。
export function AuthGate({
  shouldShow,
  showSpinnerWhen,
  redirectPath,
  withNext = false,
  showForbidden = false,
  children,
}: AuthGateProps) {
  const { hasHydrated, isAuthenticated, isAdmin } = useAuthGuard({
    shouldRedirect: (isAuth, isAdm) => !shouldShow(isAuth, isAdm),
    redirectPath,
    withNext,
  })

  if (!hasHydrated) {
    return <AppShellSkeleton />
  }

  if (!shouldShow(isAuthenticated, isAdmin)) {
    if (showForbidden && isAuthenticated) {
      return <ForbiddenState />
    }
    const showSpinner = showSpinnerWhen?.(isAuthenticated, isAdmin) ?? false
    if (showSpinner) {
      return <AppShellSkeleton />
    }
    return null
  }

  return <>{children}</>
}
