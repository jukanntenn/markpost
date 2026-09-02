import { request } from './base'
import type { MeRetention } from '@/types/users'

// MRFC 2026-09-02-user-facing-retention-visibility: the /me namespace opens
// with the caller's effective retention policy.
export const meApi = {
  retention: () => request<MeRetention>('/api/v1/me/retention'),
}
