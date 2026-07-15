"use client";

import { useState } from "react";
import { useDebouncedValue } from "@/hooks/useDebouncedValue";
import { useAdminQuery } from "@/hooks/useAdminQuery";
import type { UseQueryOptions } from "@tanstack/react-query";
import type { Paginated } from "@/types/pagination";

export type SearchOptions<TItem> = {
  queryKeyBuilder: (search: string) => readonly unknown[];
  queryFn: (search: string) => Promise<Paginated<TItem>>;
  debounceMs?: number;
};

export function useAdminSearchQuery<TItem>(
  options: Omit<UseQueryOptions<Paginated<TItem>>, "select" | "queryKey" | "queryFn"> &
    SearchOptions<TItem>,
) {
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebouncedValue(search, options.debounceMs ?? 300);

  const { items, ...query } = useAdminQuery({
    queryKey: options.queryKeyBuilder(debouncedSearch),
    queryFn: () => options.queryFn(debouncedSearch),
  });

  return { items, search, setSearch, ...query };
}
