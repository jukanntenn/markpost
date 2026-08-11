import type { Paginated } from './pagination'

export interface FeishuConfiguration {
  webhook_url: string
  card_link_url: string
}

export type ChannelConfiguration = FeishuConfiguration

export interface DeliveryChannel {
  id: number
  kind: string
  name: string
  enabled: boolean
  configuration: ChannelConfiguration
  keywords: string
  created_at: string
  updated_at: string
}

export interface DeliveryChannelsResponse {
  items: DeliveryChannel[]
}

export interface DeliveryChannelResponse {
  channel: DeliveryChannel
}

export interface AdminChannel {
  id: number
  name: string
  kind: string
  enabled: boolean
  user_id: number
  username: string
  configuration: ChannelConfiguration
  created_at: string
}

export type AdminChannelsResponse = Paginated<AdminChannel>

export type DeliveryStatus = 'delivered' | 'failed' | 'expired' | 'pending'

export interface DeliveryHistoryItem {
  id: number
  status: DeliveryStatus
  last_error: string
  created_at: string
  channel_id: number | null
  post_title: string | null
  post_qid: string | null
  channel_name: string | null
  username: string | null
}

export type DeliveryHistoryResponse = Paginated<DeliveryHistoryItem>

export interface LatestDeliveryResponse {
  items: DeliveryHistoryItem[]
}

// B2.7/K.2: delivery stats — today counters for the pipeline status bar plus
// the per-day trend for the chart.
export interface TodayCounts {
  delivered: number
  failed: number
  pending: number
}

export interface DailyStat {
  day: string // YYYY-MM-DD
  delivered: number
  failed: number
  expired: number
}

export interface DeliveryStatsResponse {
  today: TodayCounts
  trend: DailyStat[]
}

// K.2: in-flight attempts (activity feed "投递中" state).
export interface PendingAttempt {
  id: number
  post_id: number
  channel_id: number
  post_title: string
  post_qid: string
  channel_name: string
  created_at: string
}

export interface PendingAttemptsResponse {
  items: PendingAttempt[]
}

// D2.1/K.7: channels flagged by the failing-channel query.
export interface LockedChannel {
  channel_id: number
  channel_name: string
  username: string
  fails: number
  total: number
  failure_rate: number
  last_error: string
  last_at: string | null
}

export interface AdminLockedChannelsResponse {
  items: LockedChannel[]
}

export interface CreateChannelPayload {
  kind: string
  name: string
  configuration: ChannelConfiguration
  keywords?: string
}

export interface UpdateChannelPayload {
  name?: string
  configuration?: ChannelConfiguration
  keywords?: string
  enabled?: boolean
}
