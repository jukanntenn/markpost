import { useState } from 'react'
import { authApi } from '@/lib/api'

const OAUTH_STATE_KEY = 'oauth_state'
const OAUTH_NEXT_KEY = 'oauth_next'

// useGitHubOAuth implements the same-page redirect OAuth flow (auth.md §3.1-3.3).
// startOAuth fetches {url, state} from the backend, stores the expected state
// in sessionStorage for the callback page's second-layer check, then navigates
// the whole page to GitHub (no popup). The popup model is deliberately rejected
// (auth.md §3.1) — same-page redirect avoids popup blockers, cross-window
// messaging, and mobile UX problems.
export function useGitHubOAuth() {
  const [loading, setLoading] = useState(false)

  // B1.2 #4 intended-URL：发起时把 next 存入 sessionStorage，回调页登录
  // 成功后跳回（K.3 safeNext 把关）。
  const startOAuth = async (next?: string) => {
    setLoading(true)
    try {
      const { url, state } = await authApi.getOAuthUrl()
      sessionStorage.setItem(OAUTH_STATE_KEY, state)
      if (next) sessionStorage.setItem(OAUTH_NEXT_KEY, next)
      window.location.href = url
    } catch (err) {
      setLoading(false)
      throw err
    }
  }

  return { startOAuth, loading }
}

// getExpectedOAuthState returns (and clears) the state stored before the
// redirect, used by the callback page for the front-end second-layer state
// check (the backend is the primary defense; auth.md §3.3).
export function consumeExpectedOAuthState(): string | null {
  const state = sessionStorage.getItem(OAUTH_STATE_KEY)
  sessionStorage.removeItem(OAUTH_STATE_KEY)
  return state
}

// consumeOAuthNext returns (and clears) the intended-URL stored before the
// GitHub redirect (B1.2 #4). Falls back to null so the callback page lands on
// /dashboard.
export function consumeOAuthNext(): string | null {
  const next = sessionStorage.getItem(OAUTH_NEXT_KEY)
  sessionStorage.removeItem(OAUTH_NEXT_KEY)
  return next
}
