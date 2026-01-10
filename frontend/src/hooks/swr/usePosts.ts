import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";
import type { PostsPaginatedResponse } from "../../types/posts";

interface UsePostsOptions {
  refreshInterval?: number;
}

export function usePosts(page: number, limit: number = 20, options?: UsePostsOptions) {
  return useSWR<PostsPaginatedResponse>(
    page ? `/api/posts?page=${page}&limit=${limit}` : null,
    authFetcher,
    {
      refreshInterval: options?.refreshInterval,
      refreshWhenHidden: false,
      revalidateOnFocus: false,
    }
  );
}
