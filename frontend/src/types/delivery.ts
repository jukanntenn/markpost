import type { Paginated } from "./pagination";

export interface DeliveryChannel {
  id: number;
  kind: string;
  name: string;
  enabled: boolean;
  webhook_url: string;
  keywords: string;
  created_at: string;
  updated_at: string;
}

export interface DeliveryChannelsResponse {
  channels: DeliveryChannel[];
}

export interface DeliveryChannelResponse {
  channel: DeliveryChannel;
}

export interface AdminChannel {
  id: number;
  name: string;
  kind: string;
  enabled: boolean;
  user_id: number;
  webhook_url: string;
  created_at: string;
}

export type AdminChannelsResponse = Paginated<AdminChannel, "channels">;

export interface CreateChannelPayload {
  kind: string;
  name: string;
  webhook_url: string;
  keywords?: string;
}

export interface UpdateChannelPayload {
  name?: string;
  webhook_url?: string;
  keywords?: string;
  enabled?: boolean;
}
