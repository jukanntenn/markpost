import type { UseQueryOptions } from "@tanstack/react-query";
import { usePaginatedQuery } from "@/hooks/usePaginatedQuery";
import type { Paginated } from "@/types/pagination";

// useAdminQuery adapts the flat Paginated<TItem> response (items + total/page/
// limit/total_pages at the top level) into the { items, pagination } shape the
// admin table pages consume. The response is already flat (api-design.md §4).
export function useAdminQuery<TItem>(
  options: Omit<UseQueryOptions<Paginated<TItem>>, "select">,
) {
  return usePaginatedQuery<TItem, Paginated<TItem>>({
    ...options,
    extractData: (res) => ({
      items: res.items,
      pagination: { page: res.page, limit: res.limit, total: res.total, total_pages: res.total_pages },
    }),
  });
}
