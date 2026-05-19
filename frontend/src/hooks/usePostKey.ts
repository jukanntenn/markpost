"use client";

import { useQuery } from "@tanstack/react-query";
import { authApi, postKeyKeys } from "@/lib/api";

export function usePostKey() {
  return useQuery({
    queryKey: postKeyKeys.detail(),
    queryFn: authApi.queryPostKey,
  });
}
