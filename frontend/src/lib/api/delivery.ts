import { request, paginationParams } from "./base";
import type { DeliveryChannelsResponse, DeliveryChannelResponse, CreateChannelPayload, UpdateChannelPayload, DeliveryHistoryResponse, LatestDeliveryResponse } from "@/types/delivery";

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

  test: (id: number) =>
    request<{ message: string }>(`/api/v1/delivery/channels/${id}/test`, {
      method: "POST",
    }),

  listHistory: (page: number, limit: number, channelId?: number) =>
    request<DeliveryHistoryResponse>("/api/v1/delivery/history", {
      params: { ...paginationParams(page, limit), ...(channelId ? { channel_id: channelId } : {}) },
    }),

  latestPerChannel: () =>
    request<LatestDeliveryResponse>("/api/v1/delivery/latest"),
};
