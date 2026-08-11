'use client'

import { useEffect, useRef } from 'react'
import { useSearchParams, useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { useAuthStore } from '@/stores/auth'
import { authApi, ApiError, ApiErrorCodes } from '@/lib/api'
import {
  consumeExpectedOAuthState,
  consumeOAuthNext,
} from '@/hooks/useGitHubOAuth'
import { Spinner } from '@/components/ui/spinner'
import { safeNext } from '@/utils/safe-next'

// LoginCallbackPage handles the same-page OAuth redirect callback (auth.md §7,
// B1 场景 B2)。失败路径带 ?error=<code> 回 /login 显示（错误透传）；
// 用户拒绝授权（access_denied）→ 静默 /login 不显示错误。
export default function LoginCallbackPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const t = useTranslations('loginCallback')
  const setAuth = useAuthStore((state) => state.setAuth)
  const processing = useRef(false)

  useEffect(() => {
    if (processing.current) return
    processing.current = true

    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')

    // GitHub 返回 error：access_denied = 用户主动取消，静默回登录页。
    if (error) {
      router.replace(
        error === 'access_denied'
          ? '/login'
          : `/login?error=${encodeURIComponent(error)}`,
      )
      return
    }

    if (!code || !state) {
      router.replace('/login?error=missing_code')
      return
    }

    // 前端二次校验 state（后端是主防线）。
    const expectedState = consumeExpectedOAuthState()
    if (state !== expectedState) {
      router.replace('/login?error=invalid_state')
      return
    }

    authApi
      .loginWithGitHub(code, state)
      .then((data) => {
        setAuth(data.token, data.user, data.refresh_token)
        // B1.2 #4 intended-URL：登录成功跳回 OAuth 发起前所在页。
        router.replace(safeNext(consumeOAuthNext()))
      })
      .catch((err: unknown) => {
        // B1 场景 B2：错误码透传到登录页展示。
        const apiErr = err instanceof ApiError ? err : null
        if (apiErr?.code === ApiErrorCodes.Timeout) {
          router.replace('/login?error=timeout')
          return
        }
        router.replace(
          `/login?error=${encodeURIComponent(apiErr?.code ?? 'unknown')}`,
        )
      })
  }, [searchParams, router, setAuth])

  return (
    <div className="flex justify-center pt-10">
      <div className="flex flex-col items-center gap-2 text-center text-sm text-muted-foreground">
        <Spinner className="size-5" />
        <div>{t('loading')}</div>
      </div>
    </div>
  )
}
