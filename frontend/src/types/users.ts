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
