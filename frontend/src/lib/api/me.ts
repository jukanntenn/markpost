import { request } from './base'
import type { MeRetention } from '@/types/users'
import type { User } from '@/types/auth'

// The /me namespace (MRFC 2026-09-02-user-facing-retention-visibility) owns
// caller-scoped reads: the full profile and the effective retention policy.
export const meApi = {
  profile: () => request<User>('/api/v1/me'),
  retention: () => request<MeRetention>('/api/v1/me/retention'),
}
