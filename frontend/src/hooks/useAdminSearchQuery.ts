'use client'

import { useState, useCallback } from 'react'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { useAdminQuery } from '@/hooks/useAdminQuery'
import type { UseQueryOptions } from '@tanstack/react-query'
import type { Paginated } from '@/types/pagination'

export type SearchOptions<TItem> = {
  queryKeyBuilder: (search: string) => readonly unknown[]
  queryFn: (
    search: string,
    page?: number,
    limit?: number
  ) => Promise<Paginated<TItem>>
  debounceMs?: number
}

export function useAdminSearchQuery<TItem>(
  options: Omit<
    UseQueryOptions<Paginated<TItem>>,
    'select' | 'queryKey' | 'queryFn'
  > &
    SearchOptions<TItem>
) {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebouncedValue(search, options.debounceMs ?? 300)

  const handlePageChange = useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  const { items, pagination, ...query } = useAdminQuery({
    queryKey: [...options.queryKeyBuilder(debouncedSearch), { page }],
    queryFn: () => options.queryFn(debouncedSearch, page),
  })

  return {
    items,
    search,
    setSearch,
    pagination,
    page,
    onPageChange: handlePageChange,
    ...query,
  }
}
