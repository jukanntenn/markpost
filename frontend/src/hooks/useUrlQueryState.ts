'use client'

import { useCallback, useMemo } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'

// H.6 状态边界：列表筛选/分页必须进 URL query（可分享、书签、刷新保持）。
// 返回受控的分页 + 筛选状态，以及同步 URL 的 setter。
export function useUrlQueryState<T extends Record<string, string>>(
  defaults: T,
) {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()

  const state = useMemo(() => {
    const out: Record<string, string> = { ...defaults }
    for (const [k, v] of searchParams.entries()) {
      if (k in defaults) out[k] = v
    }
    return out as T
  }, [searchParams, defaults])

  const setState = useCallback(
    (patch: Partial<T>, resetPage = true) => {
      const next = { ...state, ...patch }
      const params = new URLSearchParams()
      for (const [k, v] of Object.entries(next)) {
        if (v && v !== defaults[k]) params.set(k, v)
      }
      if (resetPage) params.set('page', '1')
      const qs = params.toString()
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false })
    },
    [router, pathname, state, defaults],
  )

  const setPage = useCallback(
    (page: number) => {
      const params = new URLSearchParams(searchParams.toString())
      if (page <= 1) params.delete('page')
      else params.set('page', String(page))
      const qs = params.toString()
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false })
    },
    [router, pathname, searchParams],
  )

  return { state, setState, setPage }
}
