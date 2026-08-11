import type { Paginated } from './pagination'

// Audit log row as returned by GET /admin/audit-logs (D4.1: actor_username
// joined at read time; DEV-1: target_username joined for user-targeted rows).
export interface AuditLogItem {
  id: number
  actor_id: number
  actor_username: string
  action: string
  target_type: string
  target_id: string
  target_username: string | null
  metadata: Record<string, unknown> | null
  ip: string
  created_at: string
}

export interface AuditLogsResponse extends Paginated<AuditLogItem> {
  // D4.3 筛选计数：当前筛选下的 action 计数。
  facets: Record<string, number>
}

// D4.3 audit list query filters (URL-synced).
export interface AuditFilters {
  actor_id?: number
  action?: string
  target_type?: string
  target_id?: string
  since?: string
  until?: string
}
