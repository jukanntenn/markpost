import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";

export interface DeliveryChannel {
  id: number;
  kind: string;
  name: string;
  enabled: boolean;
  webhook_url: string;
  created_at: string;
  updated_at: string;
}

export interface DeliveryChannelsResponse {
  channels: DeliveryChannel[];
}

export function useDeliveryChannels() {
  return useSWR<DeliveryChannelsResponse>("/api/delivery/channels", authFetcher);
}
