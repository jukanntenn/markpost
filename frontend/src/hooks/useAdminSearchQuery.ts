"use client";

import { useState } from "react";
import { useDebouncedValue } from "@/hooks/useDebouncedValue";
import { useAdminQuery } from "@/hooks/useAdminQuery";
import type { UseQueryOptions } from "@tanstack/react-query";
import type { Paginated } from "@/types/pagination";

export type SearchOptions<TItem, TKey extends string> = {
  itemKey: TKey;
  queryKeyBuilder: (search: string) => readonly unknown[];
  queryFn: (search: string) => Promise<Paginated<TItem, TKey>>;
  debounceMs?: number;
};

export function useAdminSearchQuery<TItem, TKey extends string>(
  options: Omit<UseQueryOptions<Paginated<TItem, TKey>>, "select" | "queryKey" | "queryFn"> &
    SearchOptions<TItem, TKey>,
) {
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebouncedValue(search, options.debounceMs ?? 300);

  const { items, ...query } = useAdminQuery({
    queryKey: options.queryKeyBuilder(debouncedSearch),
    queryFn: () => options.queryFn(debouncedSearch),
    itemKey: options.itemKey,
  });

  return { items, search, setSearch, ...query };
}