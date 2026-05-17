"use client";

import { useQuery } from "@tanstack/react-query";
import { authApi } from "@/lib/api";

export function usePostKey() {
  return useQuery({
    queryKey: ["postKey"],
    queryFn: () => authApi.queryPostKey(),
  });
}
