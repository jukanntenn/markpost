"use client";

import { useQuery } from "@tanstack/react-query";
import { request } from "@/lib/api";
import type { PostKeyResponse } from "@/types/posts";

export function usePostKey() {
  return useQuery<PostKeyResponse>({
    queryKey: ["postKey"],
    queryFn: () => request<PostKeyResponse>("/api/v1/post_key"),
  });
}
