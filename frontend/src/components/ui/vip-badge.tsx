import { Badge } from '@/components/ui/badge'

// Locale-invariant honorific mark beside usernames (MRFC
// 2026-08-23-vip-badge-and-admin-management). Purely cosmetic: vip grants no
// permissions.
export function VipBadge() {
  return <Badge variant="accent">VIP</Badge>
}
