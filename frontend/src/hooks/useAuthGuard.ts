'use client'

import { useEffect, useRef } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { useAuthReady } from '@/hooks/useAuthReady'
import { useAuthStore } from '@/stores/auth'

interface AuthGuardOptions {
  shouldRedirect: (isAuthenticated: boolean, isAdmin: boolean) => boolean
  redirectPath: string
  // K.3 intended-URL：redirect 时携带 ?next= 当前路径（防 open redirect 由
  // 登录页 safeNext 把关）
  withNext?: boolean
}

export function useAuthGuard({
  shouldRedirect,
  redirectPath,
  withNext = false,
}: AuthGuardOptions) {
  const router = useRouter()
  const pathname = usePathname()
  const sessionExpired = useAuthStore((s) => s.sessionExpired)
  const clearSessionExpired = useAuthStore((s) => s.clearSessionExpired)
  const { hasHydrated, isAuthenticated, isAdmin } = useAuthReady()

  // BUG-4: the redirect must fire exactly once. clearSessionExpired() flips
  // sessionExpired (a dep) and AuthGate passes a fresh inline shouldRedirect
  // every render (also a dep), so the effect re-ran AFTER the reason had been
  // appended and built a second, reason-less URL that clobbered the first via
  // router.replace. The lock makes the redirect one-shot within this mount.
  const redirectedRef = useRef(false)

  useEffect(() => {
    if (redirectedRef.current) return
    if (hasHydrated && shouldRedirect(isAuthenticated, isAdmin)) {
      let target =
        withNext && pathname
          ? `${redirectPath}?next=${encodeURIComponent(pathname)}`
          : redirectPath
      // B1.8 场景 D：refresh 失败后跳登录页需带 ?reason=session_expired。
      if (sessionExpired) {
        target = `${target}${target.includes('?') ? '&' : '?'}reason=session_expired`
        clearSessionExpired()
      }
      redirectedRef.current = true
      router.replace(target)
    }
  }, [
    hasHydrated,
    isAuthenticated,
    isAdmin,
    pathname,
    router,
    sessionExpired,
    clearSessionExpired,
    shouldRedirect,
    redirectPath,
    withNext,
  ])

  return { hasHydrated, isAuthenticated, isAdmin }
}
