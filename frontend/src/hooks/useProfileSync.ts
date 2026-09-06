'use client'

import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { meApi, meKeys } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

// The persisted auth store holds a login-time user snapshot; fields an admin
// can flip afterwards (vip, role, ban state) would stay stale across page
// refreshes until re-login. Each full page load remounts the app with an empty
// query cache, so this fetch runs once per load and re-syncs the snapshot.
export function useProfileSync() {
  const hasHydrated = useAuthStore((state) => state._hasHydrated)
  const token = useAuthStore((state) => state.token)
  const setUser = useAuthStore((state) => state.setUser)

  const query = useQuery({
    queryKey: meKeys.profile(),
    queryFn: () => meApi.profile(),
    enabled: hasHydrated && !!token,
  })

  useEffect(() => {
    if (!query.data) return
    // A late response can land after logout() nulled the session; applying it
    // would resurrect a stale user alongside empty tokens.
    if (useAuthStore.getState().token) {
      setUser(query.data)
    }
  }, [query.data, setUser])
}
