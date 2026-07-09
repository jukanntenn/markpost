import { request, paginationParams } from "./base";
import type { AdminUsersResponse } from "@/types/users";
import type { AdminPostsResponse } from "@/types/posts";
import type { AdminChannelsResponse, DeliveryHistoryResponse } from "@/types/delivery";

export const adminApi = {
  listUsers: (page?: number, limit?: number) =>
    request<AdminUsersResponse>("/api/v1/admin/users", {
      params: paginationParams(page, limit),
    }),

  listPosts: (search?: string, page?: number, limit?: number) =>
    request<AdminPostsResponse>("/api/v1/admin/posts", {
      params: { ...(search && { search }), ...paginationParams(page, limit) },
    }),

  listChannels: (page?: number, limit?: number) =>
    request<AdminChannelsResponse>("/api/v1/admin/channels", {
      params: paginationParams(page, limit),
    }),

  listDeliveryHistory: (page?: number, limit?: number) =>
    request<DeliveryHistoryResponse>("/api/v1/admin/delivery-history", {
      params: paginationParams(page, limit),
    }),
};
