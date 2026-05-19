import type { UseQueryOptions } from "@tanstack/react-query";
import { usePaginatedQuery } from "@/hooks/usePaginatedQuery";
import type { Paginated } from "@/types/pagination";

export function useAdminQuery<TItem, TKey extends string>(
  options: Omit<UseQueryOptions<Paginated<TItem, TKey>>, "select"> & {
    itemKey: TKey;
  },
) {
  const { itemKey, ...queryOptions } = options;
  return usePaginatedQuery<TItem, Paginated<TItem, TKey>>({
    ...queryOptions,
    extractData: (res) => ({
      items: res[itemKey],
      pagination: res.pagination,
    }),
  });
}
