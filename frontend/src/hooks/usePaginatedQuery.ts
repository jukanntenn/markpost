import { useQuery, UseQueryOptions } from "@tanstack/react-query";
import type { Pagination } from "@/types/pagination";

export function usePaginatedQuery<TItem, TResponse>(
  options: Omit<UseQueryOptions<TResponse>, "select"> & {
    extractData: (response: TResponse) => { items: TItem[]; pagination: Pagination | undefined };
  },
) {
  const { extractData, ...queryOptions } = options;
  const result = useQuery<TResponse>(queryOptions);
  const data = result.data ? extractData(result.data) : { items: [], pagination: undefined };
  return {
    ...result,
    items: data.items,
    pagination: data.pagination,
  };
}
