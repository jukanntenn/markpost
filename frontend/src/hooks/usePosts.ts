"use client";

import { useQuery } from "@tanstack/react-query";
import { request } from "@/lib/api";
import type { PostsPaginatedResponse } from "@/types/posts";

interface UsePostsOptions {
  refetchInterval?: number;
}

export function usePosts(page: number, limit: number = 20, options?: UsePostsOptions) {
  return useQuery<PostsPaginatedResponse>({
    queryKey: ["posts", page, limit],
    queryFn: () => request<PostsPaginatedResponse>(`/api/v1/posts?page=${page}&limit=${limit}`),
    refetchInterval: options?.refetchInterval,
    refetchOnWindowFocus: false,
  });
}
