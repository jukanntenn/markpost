import { request } from './base'
import type {
  PostKeyResponse,
  LoginResponse,
  RefreshResponse,
  OAuthUrlResponse,
  LogoutResponse,
  ChangePasswordResponse,
  RotatePostKeyResponse,
  SessionsResponse,
} from '@/types/auth'

export const authApi = {
  login: (username: string, password: string) =>
    request<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      json: { username, password },
      skipAuthRefresh: true,
      // I.2: 登录属低频关键请求，超时放宽到 30s（默认）。
    }),

  loginWithGitHub: (code: string, state: string) =>
    request<LoginResponse>('/api/v1/oauth/login', {
      method: 'POST',
      json: { code, state },
      skipAuthRefresh: true,
      timeoutMs: 15_000, // I.2: OAuth 回调 15s
    }),

  getOAuthUrl: () =>
    request<OAuthUrlResponse>('/api/v1/oauth/url', {
      skipAuthRefresh: true,
      timeoutMs: 10_000, // I.2: OAuth 发起 10s
    }),

  logout: () =>
    request<LogoutResponse>('/api/v1/auth/logout', {
      method: 'POST',
    }),

  refreshToken: (refreshToken: string) =>
    request<RefreshResponse>('/api/v1/auth/refresh', {
      method: 'POST',
      json: { refresh_token: refreshToken },
      skipAuthRefresh: true,
    }),

  // C2.2: 成功后返回新 token 对，前端 setTokens 无缝继续。
  changePassword: (currentPassword: string, newPassword: string) =>
    request<ChangePasswordResponse>('/api/v1/auth/change-password', {
      method: 'POST',
      json: {
        current_password: currentPassword,
        new_password: newPassword,
      },
    }),

  queryPostKey: () => request<PostKeyResponse>('/api/v1/post-key'),

  rotatePostKey: () =>
    request<RotatePostKeyResponse>('/api/v1/post-key/rotate', {
      method: 'POST',
    }),

  // I.12: 用户透明 —— 自己的会话。
  listSessions: () => request<SessionsResponse>('/api/v1/auth/sessions'),

  revokeSession: (tokenId: number) =>
    request<void>(`/api/v1/auth/sessions/${tokenId}`, { method: 'DELETE' }),

  revokeAllSessions: () =>
    request<void>('/api/v1/auth/sessions', { method: 'DELETE' }),
}
