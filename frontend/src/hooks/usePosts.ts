"use client";

import { useQuery } from "@tanstack/react-query";
import { authFetcher } from "@/lib/api/fetcher";
import type { PostsPaginatedResponse } from "@/types/posts";

interface UsePostsOptions {
  refetchInterval?: number;
}

export function usePosts(page: number, limit: number = 20, options?: UsePostsOptions) {
  return useQuery<PostsPaginatedResponse>({
    queryKey: ["posts", page, limit],
    queryFn: () => authFetcher(`/api/posts?page=${page}&limit=${limit}`),
    refetchInterval: options?.refetchInterval,
    refetchOnWindowFocus: false,
  });
}
