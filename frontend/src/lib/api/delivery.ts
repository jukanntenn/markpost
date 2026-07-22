import { request, paginationParams } from "./base";
import type { DeliveryChannelsResponse, DeliveryChannelResponse, CreateChannelPayload, UpdateChannelPayload, DeliveryHistoryResponse } from "@/types/delivery";

export const deliveryApi = {
  list: () =>
    request<DeliveryChannelsResponse>("/api/v1/delivery/channels"),

  create: (data: CreateChannelPayload) =>
    request<DeliveryChannelResponse>("/api/v1/delivery/channels", {
      method: "POST",
      json: data,
    }),

  update: (id: number, data: UpdateChannelPayload) =>
    request<DeliveryChannelResponse>(`/api/v1/delivery/channels/${id}`, {
      method: "PATCH",
      json: data,
    }),

  delete: (id: number) =>
    request<{ message: string }>(`/api/v1/delivery/channels/${id}`, {
      method: "DELETE",
    }),

  listHistory: (page: number, limit: number) =>
    request<DeliveryHistoryResponse>("/api/v1/delivery/history", {
      params: paginationParams(page, limit),
    }),
};
