"use client";

import { postsApi, postKeys } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/constants";
import { useAdminQuery } from "@/hooks/useAdminQuery";
import type { PostListItem } from "@/types/posts";

interface UsePostsOptions {
  refetchInterval?: number;
}

export function usePosts(page: number, limit: number = DEFAULT_PAGE_SIZE, options?: UsePostsOptions) {
  const { items, ...rest } = useAdminQuery<PostListItem>({
    queryKey: postKeys.list(page, limit),
    queryFn: () => postsApi.list(page, limit),
    refetchInterval: options?.refetchInterval,
    refetchOnWindowFocus: false,
  });

  return {
    posts: items,
    ...rest,
  };
}
