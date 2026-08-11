import { request, paginationParams } from './base'
import type {
  AdminUsersResponse,
  AdminUser,
  AdminResetPasswordResponse,
  AdminStatsResponse,
} from '@/types/users'
import type { AdminPostsResponse } from '@/types/posts'
import type {
  AdminChannelsResponse,
  DeliveryHistoryResponse,
  AdminLockedChannelsResponse,
  DeliveryStatsResponse,
} from '@/types/delivery'
import type { AuditLogsResponse, AuditFilters } from '@/types/audit'
import type { SessionsResponse } from '@/types/auth'

export const adminApi = {
  // D3.1: username LIKE search.
  listUsers: (page?: number, search?: string, limit?: number) =>
    request<AdminUsersResponse>('/api/v1/admin/users', {
      params: {
        ...(search ? { search } : {}),
        ...paginationParams(page, limit),
      },
    }),

  // D3.2: 用户详情（资料区数据：post_key/last_login_at）。
  getUser: (id: number) => request<AdminUser>(`/api/v1/admin/users/${id}`),

  createUser: (data: { email: string; username: string; password: string }) =>
    request<AdminUser>('/api/v1/admin/users', {
      method: 'POST',
      json: data,
    }),

  setUserRole: (id: number, role: string) =>
    request<AdminUser>(`/api/v1/admin/users/${id}/role`, {
      method: 'PATCH',
      json: { role },
    }),

  // D3.3 方案 B: 无请求体，系统生成临时密码一次性返回。
  resetUserPassword: (id: number) =>
    request<AdminResetPasswordResponse>(`/api/v1/admin/users/${id}/password`, {
      method: 'POST',
    }),

  setUserActive: (id: number, active: boolean) =>
    request<AdminUser>(`/api/v1/admin/users/${id}/active`, {
      method: 'PATCH',
      json: { active },
    }),

  deleteUser: (id: number) =>
    request<void>(`/api/v1/admin/users/${id}`, { method: 'DELETE' }),

  // F.9: title search + username filter.
  listPosts: (
    page?: number,
    search?: string,
    username?: string,
    limit?: number,
  ) =>
    request<AdminPostsResponse>('/api/v1/admin/posts', {
      params: {
        ...(search ? { search } : {}),
        ...(username ? { username } : {}),
        ...paginationParams(page, limit),
      },
    }),

  deletePost: (qid: string) =>
    request<void>(`/api/v1/admin/posts/${qid}`, { method: 'DELETE' }),

  listChannels: (page?: number, limit?: number) =>
    request<AdminChannelsResponse>('/api/v1/admin/delivery/channels', {
      params: paginationParams(page, limit),
    }),

  setChannelEnabled: (id: number, enabled: boolean) =>
    request<void>(`/api/v1/admin/delivery/channels/${id}/enabled`, {
      method: 'PATCH',
      json: { enabled },
    }),

  deleteChannel: (id: number) =>
    request<void>(`/api/v1/admin/delivery/channels/${id}`, {
      method: 'DELETE',
    }),

  // F.8: user/channel/status filters.
  listDeliveryHistory: (
    page?: number,
    filter?: { user_id?: number; channel_id?: number; status?: string },
    limit?: number,
  ) =>
    request<DeliveryHistoryResponse>('/api/v1/admin/delivery/history', {
      params: {
        ...(filter?.user_id ? { user_id: filter.user_id } : {}),
        ...(filter?.channel_id ? { channel_id: filter.channel_id } : {}),
        ...(filter?.status && filter.status !== 'all'
          ? { status: filter.status }
          : {}),
        ...paginationParams(page, limit),
      },
    }),

  // D4: audit list with filters + facets (筛选计数).
  listAuditLogs: (page?: number, filters?: AuditFilters, limit?: number) =>
    request<AuditLogsResponse>('/api/v1/admin/audit-logs', {
      params: {
        ...(filters?.actor_id ? { actor_id: filters.actor_id } : {}),
        ...(filters?.action && filters.action !== 'all'
          ? { action: filters.action }
          : {}),
        ...(filters?.target_type && filters.target_type !== 'all'
          ? { target_type: filters.target_type }
          : {}),
        ...(filters?.target_id ? { target_id: filters.target_id } : {}),
        ...(filters?.since ? { since: filters.since } : {}),
        ...(filters?.until ? { until: filters.until } : {}),
        ...paginationParams(page, limit),
      },
    }),

  getStats: () => request<AdminStatsResponse>('/api/v1/admin/stats'),

  // D2.5: admin 全站趋势。
  deliveryStats: (days = 7) =>
    request<DeliveryStatsResponse>('/api/v1/admin/delivery/stats', {
      params: { days },
    }),

  // D2.1: 需要关注 —— 投递持续失败渠道。
  lockedChannels: () =>
    request<AdminLockedChannelsResponse>('/api/v1/admin/locked-channels'),

  // D3.2: 用户会话。
  listSessions: (userId: number) =>
    request<SessionsResponse>(`/api/v1/admin/users/${userId}/sessions`),

  revokeAllSessions: (userId: number) =>
    request<void>(`/api/v1/admin/users/${userId}/sessions`, {
      method: 'DELETE',
    }),

  revokeSession: (tokenId: number) =>
    request<void>(`/api/v1/admin/sessions/${tokenId}`, { method: 'DELETE' }),
}
