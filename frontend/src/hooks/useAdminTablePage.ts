"use client";

import { useAdminQuery } from "@/hooks/useAdminQuery";
import { useAdminSearchQuery, type SearchOptions } from "@/hooks/useAdminSearchQuery";
import type { QueryStateProps } from "@/types/query-state";
import type { UseQueryOptions } from "@tanstack/react-query";
import type { Paginated } from "@/types/pagination";

function toQueryStateProps(
  query: { isLoading: boolean; error: Error | null },
  t: (key: string) => string,
): QueryStateProps {
  return {
    isLoading: query.isLoading,
    error: query.error,
    loadingText: t("loading"),
    errorText: t("error"),
  };
}

type AdminQueryOptions<TItem, TKey extends string> = Omit<
  UseQueryOptions<Paginated<TItem, TKey>>,
  "select"
> & { itemKey: TKey };

export function useAdminTablePage<TItem, TKey extends string>(
  options: AdminQueryOptions<TItem, TKey> & { t: (key: string) => string },
) {
  const { t, ...queryOptions } = options;
  const { items, ...query } = useAdminQuery<TItem, TKey>(queryOptions);
  return { items, ...query, ...toQueryStateProps(query, t) };
}

export function useAdminSearchTablePage<TItem, TKey extends string>(
  options: Omit<UseQueryOptions<Paginated<TItem, TKey>>, "select" | "queryKey" | "queryFn"> &
    SearchOptions<TItem, TKey> & { t: (key: string) => string },
) {
  const { t, ...queryOptions } = options;
  const { items, search, setSearch, ...query } = useAdminSearchQuery<TItem, TKey>(queryOptions);
  return { items, search, setSearch, ...query, ...toQueryStateProps(query, t) };
}
