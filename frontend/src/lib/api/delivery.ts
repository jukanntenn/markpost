import { request } from "./base";
import type { DeliveryChannelsResponse, DeliveryChannelResponse } from "@/types/delivery";

export const deliveryApi = {
  list: () =>
    request<DeliveryChannelsResponse>("/api/v1/delivery/channels").then(
      (data) => data.channels
    ),

  create: (data: {
    kind: string;
    name: string;
    webhook_url: string;
    keywords?: string;
  }) =>
    request<DeliveryChannelResponse>("/api/v1/delivery/channels", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    id: number,
    data: {
      name?: string;
      webhook_url?: string;
      keywords?: string;
      enabled?: boolean;
    }
  ) =>
    request<DeliveryChannelResponse>(`/api/v1/delivery/channels/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    request<{ message: string }>(`/api/v1/delivery/channels/${id}`, {
      method: "DELETE",
    }),
};
