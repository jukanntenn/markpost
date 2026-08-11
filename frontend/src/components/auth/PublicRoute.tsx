'use client'

import { useSearchParams } from 'next/navigation'
import { AuthGate } from '@/components/auth/AuthGate'
import { publicRoute } from '@/components/auth/route-configs'
import { safeNext } from '@/utils/safe-next'

export function PublicRoute({ children }: { children: React.ReactNode }) {
  const searchParams = useSearchParams()
  // K.3：已登录用户回 /login 时，优先回到 intended-URL（?next=）。
  const next = safeNext(searchParams.get('next'))
  return (
    <AuthGate
      shouldShow={publicRoute.shouldShow}
      redirectPath={next === '/dashboard' ? publicRoute.redirectPath : next}
      showSpinnerWhen={publicRoute.showSpinnerWhen}
    >
      {children}
    </AuthGate>
  )
}
