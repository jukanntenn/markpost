export type UserRole = 'admin' | 'user'

export interface User {
  id: number
  email: string
  username: string
  name?: string
  avatar_url?: string | null
  role?: UserRole
  // B1.12: ban-state awareness
  is_active?: boolean
  is_email_verified?: boolean
}

export interface LoginResponse {
  token: string
  refresh_token: string
  expires_in: number
  user: User
}

export interface RefreshResponse {
  token: string
  refresh_token: string
  expires_in: number
}

// C2.2: change-password success body — a fresh token pair, no re-login.
export interface ChangePasswordResponse {
  token: string
  refresh_token: string
  expires_in: number
}

export interface OAuthUrlResponse {
  url: string
  state: string
}

export interface LogoutResponse {
  message: string
}

export interface PostKeyResponse {
  post_key: string
  created_at: string
}

// C2.5: post-key rotation returns the new key.
export interface RotatePostKeyResponse {
  post_key: string
}

// RefreshToken row as exposed by GET /auth/sessions (I.12) and
// GET /admin/users/:id/sessions (D3.2). No IP/UA/device info exists.
export interface Session {
  id: number
  user_id: number
  token_hash: string
  revoked: boolean
  expires_at: string
  created_at: string
}

export interface SessionsResponse {
  sessions: Session[]
}
