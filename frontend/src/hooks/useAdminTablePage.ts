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

type AdminQueryOptions<TItem> = Omit<UseQueryOptions<Paginated<TItem>>, "select">;

export function useAdminTablePage<TItem>(
  options: AdminQueryOptions<TItem> & { t: (key: string) => string },
) {
  const { t, ...queryOptions } = options;
  const { items, ...query } = useAdminQuery<TItem>(queryOptions);
  return { items, ...query, ...toQueryStateProps(query, t) };
}

export function useAdminSearchTablePage<TItem>(
  options: Omit<UseQueryOptions<Paginated<TItem>>, "select" | "queryKey" | "queryFn"> &
    SearchOptions<TItem> & { t: (key: string) => string },
) {
  const { t, ...queryOptions } = options;
  const { items, search, setSearch, ...query } = useAdminSearchQuery<TItem>(queryOptions);
  return { items, search, setSearch, ...query, ...toQueryStateProps(query, t) };
}
