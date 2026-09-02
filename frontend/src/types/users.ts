import type { UserRole } from './auth'
import type { Paginated } from './pagination'

// AdminUser: list item + detail profile data (D3.1/D3.2). post_key and
// last_login_at are exposed for the detail page; the frontend masks the key.
export interface AdminUser {
  id: number
  username: string
  name: string
  email: string
  is_email_verified: boolean
  github_id: number | null
  role: UserRole
  is_active: boolean
  vip: boolean
  // Per-user history retention policy (MRFC
  // 2026-08-31-per-user-history-retention-policy): null = inherit the global
  // default, 0 = keep forever, 1-3650 = keep N days.
  retention_days: number | null
  post_key: string
  last_login_at: string | null
  created_at: string
}

export type AdminUsersResponse = Paginated<AdminUser>

// D3.3 方案 B: reset-password returns the generated temporary password once.
export interface AdminResetPasswordResponse {
  password: string
}

// D2.4: admin overview stats with week deltas.
export interface AdminStats {
  users: number
  posts: number
  channels: number
  history: number
  banned_users: number
  users_week_delta: number
  posts_week_delta: number
  history_week_delta: number
}

export interface AdminStatsResponse {
  counts: AdminStats
}

// Runtime settings row (GET/PUT /api/v1/admin/settings). v1 carries the
// GitHub-login VIP strategy switch under key "vip".
export interface AdminSettingItem {
  key: string
  // The vip key owns {enabled}; vip_retention_days owns {days}
  // (null/absent = follow the global config).
  value: { enabled: boolean; days?: number | null }
  updated_by: number | null
  updated_at: string
}

// Deletion preview for a candidate retention policy (confirm dialog data).
export interface RetentionImpact {
  users_affected: number
  posts_to_delete: number
  history_to_delete: number
}

// Global fallback windows mirrored from config (renders the inherit value).
export interface RetentionDefaults {
  post_retention_days: number
}

export interface AdminSettingsResponse {
  items: AdminSettingItem[]
}
