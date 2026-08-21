import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { User } from '@/types/auth'

export interface AuthState {
  token: string | null
  refreshToken: string | null
  user: User | null
  _hasHydrated: boolean
  // B1.8 场景 D：refresh 失败 → 标记会话过期，守卫跳转 /login?reason=session_expired。
  sessionExpired: boolean

  setAuth: (token: string, user: User, refreshToken?: string) => void
  setTokens: (token: string, refreshToken: string) => void
  logout: () => void
  markSessionExpired: () => void
  clearSessionExpired: () => void
  setHasHydrated: (state: boolean) => void
}

const AUTH_STORAGE_KEY = 'markpost_auth'

export const useAuthStore = create<AuthState>()(
  persist(
    (set, _get) => ({
      token: null,
      refreshToken: null,
      user: null,
      _hasHydrated: false,
      sessionExpired: false,

      setAuth: (token, user, refreshToken) =>
        set({
          token,
          user,
          refreshToken: refreshToken || null,
          sessionExpired: false,
        }),

      setTokens: (token, refreshToken) =>
        set({ token, refreshToken, sessionExpired: false }),

      logout: () => set({ token: null, refreshToken: null, user: null }),

      markSessionExpired: () => set({ sessionExpired: true }),

      clearSessionExpired: () => set({ sessionExpired: false }),

      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: AUTH_STORAGE_KEY,
      partialize: ({ token, refreshToken, user }) => ({
        token,
        refreshToken,
        user,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true)
      },
    },
  ),
)

// Cross-tab propagation. zustand's persist reads storage once at hydration
// and does not sync tabs: without this listener every tab keeps a stale
// in-memory token pair after another tab rotates, and replaying it trips the
// backend's reuse detection (which revokes ALL sessions). Storage events fire
// only in the *other* tabs on every persist write, so this also propagates
// logout. Guarded for SSR/static prerender.
if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event) => {
    if (event.key === null || event.key === AUTH_STORAGE_KEY) {
      void useAuthStore.persist.rehydrate()
    }
  })
}
