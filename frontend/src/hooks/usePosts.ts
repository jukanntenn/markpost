"use client";

import { useQuery } from "@tanstack/react-query";
import { postsApi } from "@/lib/api";

interface UsePostsOptions {
  refetchInterval?: number;
}

export function usePosts(page: number, limit: number = 20, options?: UsePostsOptions) {
  return useQuery({
    queryKey: ["posts", page, limit],
    queryFn: () => postsApi.list(page, limit),
    refetchInterval: options?.refetchInterval,
    refetchOnWindowFocus: false,
  });
}
