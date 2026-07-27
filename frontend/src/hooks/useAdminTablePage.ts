'use client'

import { useState, useCallback } from 'react'
import { useAdminQuery } from '@/hooks/useAdminQuery'
import {
  useAdminSearchQuery,
  type SearchOptions,
} from '@/hooks/useAdminSearchQuery'
import type { QueryStateProps } from '@/types/query-state'
import type { UseQueryOptions } from '@tanstack/react-query'
import type { Paginated } from '@/types/pagination'

function toQueryStateProps(
  query: { isLoading: boolean; error: Error | null },
  t: (key: string) => string
): QueryStateProps {
  return {
    isLoading: query.isLoading,
    error: query.error,
    loadingText: t('loading'),
    errorText: t('error'),
  }
}

type AdminQueryOptions<TItem> = Omit<
  UseQueryOptions<Paginated<TItem>>,
  'select'
>

export function useAdminTablePage<TItem>(
  options: Omit<AdminQueryOptions<TItem>, 'queryFn'> & {
    t: (key: string) => string
    queryFn: (page?: number, limit?: number) => Promise<Paginated<TItem>>
    queryKeyBuilder?: (page: number) => readonly unknown[]
  }
) {
  const { t, queryFn, queryKeyBuilder, ...restOptions } = options
  const [page, setPage] = useState(1)

  const handlePageChange = useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  const { items, pagination, ...query } = useAdminQuery<TItem>({
    ...restOptions,
    queryKey: queryKeyBuilder
      ? queryKeyBuilder(page)
      : [...(restOptions.queryKey as unknown[]), { page }],
    queryFn: () => queryFn(page),
  })

  return {
    items,
    pagination,
    page,
    onPageChange: handlePageChange,
    ...query,
    ...toQueryStateProps(query, t),
  }
}

export function useAdminSearchTablePage<TItem>(
  options: Omit<
    UseQueryOptions<Paginated<TItem>>,
    'select' | 'queryKey' | 'queryFn'
  > &
    SearchOptions<TItem> & { t: (key: string) => string }
) {
  const { t, ...queryOptions } = options
  const { items, search, setSearch, pagination, page, onPageChange, ...query } =
    useAdminSearchQuery<TItem>(queryOptions)
  return {
    items,
    search,
    setSearch,
    pagination,
    page,
    onPageChange,
    ...query,
    ...toQueryStateProps(query, t),
  }
}
