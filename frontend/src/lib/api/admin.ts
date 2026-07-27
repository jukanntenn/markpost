import { request, paginationParams } from './base'
import type { AdminUsersResponse } from '@/types/users'
import type { AdminPostsResponse } from '@/types/posts'
import type {
  AdminChannelsResponse,
  DeliveryHistoryResponse,
} from '@/types/delivery'

export const adminApi = {
  listUsers: (page?: number, limit?: number) =>
    request<AdminUsersResponse>('/api/v1/admin/users', {
      params: paginationParams(page, limit),
    }),

  createUser: (data: { email: string; username: string; password: string }) =>
    request<{ id: number; username: string; email: string; role: string }>(
      '/api/v1/admin/users',
      { method: 'POST', json: data }
    ),

  setUserRole: (id: number, role: string) =>
    request<void>(`/api/v1/admin/users/${id}/role`, {
      method: 'PATCH',
      json: { role },
    }),

  resetUserPassword: (id: number, password: string) =>
    request<void>(`/api/v1/admin/users/${id}/password`, {
      method: 'POST',
      json: { password },
    }),

  setUserActive: (id: number, active: boolean) =>
    request<void>(`/api/v1/admin/users/${id}/active`, {
      method: 'PATCH',
      json: { active },
    }),

  deleteUser: (id: number) =>
    request<void>(`/api/v1/admin/users/${id}`, { method: 'DELETE' }),

  listPosts: (search?: string, page?: number, limit?: number) =>
    request<AdminPostsResponse>('/api/v1/admin/posts', {
      params: { ...(search && { search }), ...paginationParams(page, limit) },
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

  listDeliveryHistory: (page?: number, limit?: number) =>
    request<DeliveryHistoryResponse>('/api/v1/admin/delivery/history', {
      params: paginationParams(page, limit),
    }),

  getStats: () =>
    request<{
      counts: {
        users: number
        posts: number
        channels: number
        history: number
      }
    }>('/api/v1/admin/stats'),
}
