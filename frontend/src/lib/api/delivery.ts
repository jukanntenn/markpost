import { request, paginationParams } from './base'
import type {
  DeliveryChannelsResponse,
  DeliveryChannelResponse,
  CreateChannelPayload,
  UpdateChannelPayload,
  DeliveryHistoryResponse,
  LatestDeliveryResponse,
  DeliveryStatsResponse,
  PendingAttemptsResponse,
} from '@/types/delivery'

export const deliveryApi = {
  list: () => request<DeliveryChannelsResponse>('/api/v1/delivery/channels'),

  create: (data: CreateChannelPayload) =>
    request<DeliveryChannelResponse>('/api/v1/delivery/channels', {
      method: 'POST',
      json: data,
    }),

  update: (id: number, data: UpdateChannelPayload) =>
    request<DeliveryChannelResponse>(`/api/v1/delivery/channels/${id}`, {
      method: 'PATCH',
      json: data,
    }),

  delete: (id: number) =>
    request<{ message: string }>(`/api/v1/delivery/channels/${id}`, {
      method: 'DELETE',
    }),

  test: (id: number) =>
    request<{ message: string }>(`/api/v1/delivery/channels/${id}/test`, {
      method: 'POST',
      timeoutMs: 30_000, // I.2: 测试投递 30s
    }),

  // B3.4: channel_id + status filters.
  listHistory: (
    page: number,
    limit: number,
    channelId?: number,
    status?: string,
  ) =>
    request<DeliveryHistoryResponse>('/api/v1/delivery/history', {
      params: {
        ...paginationParams(page, limit),
        ...(channelId ? { channel_id: channelId } : {}),
        ...(status && status !== 'all' ? { status } : {}),
      },
    }),

  latestPerChannel: () =>
    request<LatestDeliveryResponse>('/api/v1/delivery/latest'),

  // B2.7/K.2: today counters + trend.
  stats: (days = 7) =>
    request<DeliveryStatsResponse>('/api/v1/delivery/stats', {
      params: { days },
    }),

  // K.2: in-flight attempts for the activity feed.
  pending: () => request<PendingAttemptsResponse>('/api/v1/delivery/pending'),
}
