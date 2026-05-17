import { request } from "./base";
import type { AdminUsersResponse } from "@/types/users";
import type { AdminPostsResponse } from "@/types/posts";
import type { AdminChannelsResponse } from "@/types/delivery";

export const adminApi = {
  listUsers: () =>
    request<AdminUsersResponse>("/api/v1/admin/users"),

  listPosts: (search?: string) =>
    request<AdminPostsResponse>("/api/v1/admin/posts", {
      params: search ? { search } : undefined,
    }),

  listChannels: () =>
    request<AdminChannelsResponse>("/api/v1/admin/channels"),
};
