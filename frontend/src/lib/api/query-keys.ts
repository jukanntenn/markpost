import type { QueryClient } from "@tanstack/react-query";

export function invalidateKey(queryClient: QueryClient, queryKey: readonly unknown[]) {
  return queryClient.invalidateQueries({ queryKey });
}

export const postKeys = {
  all: () => ["posts"] as const,
  list: (page: number, limit: number) => [...postKeys.all(), "list", { page, limit }] as const,
};

export const adminKeys = {
  all: () => ["admin"] as const,
  users: {
    all: () => [...adminKeys.all(), "users"] as const,
  },
  posts: {
    all: () => [...adminKeys.all(), "posts"] as const,
    list: (search: string) => [...adminKeys.posts.all(), { search }] as const,
  },
  channels: {
    all: () => [...adminKeys.all(), "channels"] as const,
  },
  history: {
    all: () => [...adminKeys.all(), "history"] as const,
  },
};

export const deliveryKeys = {
  all: () => ["delivery"] as const,
  channels: () => [...deliveryKeys.all(), "channels"] as const,
  detail: (id: number) => [...deliveryKeys.all(), "detail", id] as const,
  history: (page: number, limit: number) => [...deliveryKeys.all(), "history", { page, limit }] as const,
};

export const postKeyKeys = {
  all: () => ["postKey"] as const,
  detail: () => [...postKeyKeys.all()] as const,
};
