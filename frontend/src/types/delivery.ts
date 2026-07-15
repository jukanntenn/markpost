import type { Paginated } from "./pagination";

export interface FeishuConfiguration {
  webhook_url: string;
  card_link_url: string;
}

export type ChannelConfiguration = FeishuConfiguration;

export interface DeliveryChannel {
  id: number;
  kind: string;
  name: string;
  enabled: boolean;
  configuration: ChannelConfiguration;
  keywords: string;
  created_at: string;
  updated_at: string;
}

export type DeliveryChannelsResponse = Paginated<DeliveryChannel>;

export interface DeliveryChannelResponse {
  channel: DeliveryChannel;
}

export interface AdminChannel {
  id: number;
  name: string;
  kind: string;
  enabled: boolean;
  user_id: number;
  configuration: ChannelConfiguration;
  created_at: string;
}

export type AdminChannelsResponse = Paginated<AdminChannel>;

export type DeliveryStatus = "delivered" | "failed" | "expired";

export interface DeliveryHistoryItem {
  id: number;
  status: DeliveryStatus;
  last_error: string;
  created_at: string;
  post_title: string | null;
  post_qid: string | null;
  channel_name: string | null;
  username: string | null;
}

export type DeliveryHistoryResponse = Paginated<DeliveryHistoryItem>;

export interface CreateChannelPayload {
  kind: string;
  name: string;
  configuration: ChannelConfiguration;
  keywords?: string;
}

export interface UpdateChannelPayload {
  name?: string;
  configuration?: ChannelConfiguration;
  keywords?: string;
  enabled?: boolean;
}
