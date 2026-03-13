"use client";

import { useQuery } from "@tanstack/react-query";
import { authFetcher } from "@/lib/api/fetcher";
import type { PostKeyResponse } from "@/types/posts";

export function usePostKey() {
  return useQuery<PostKeyResponse>({
    queryKey: ["postKey"],
    queryFn: () => authFetcher("/api/post-key"),
  });
}
