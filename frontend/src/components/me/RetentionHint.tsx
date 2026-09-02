'use client'

import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { ShieldCheckIcon } from 'lucide-react'
import { meApi, meKeys } from '@/lib/api'

// MRFC 2026-09-02-user-facing-retention-visibility: one muted line stating
// the outcome of the caller's effective policy — never the mechanics. The
// value moves only when an admin acts or a deploy changes config, so the
// query carries a long staleTime instead of riding the list polling.
export function RetentionHint({ kind }: { kind: 'posts' | 'history' }) {
  const t = useTranslations('me.retention')
  const query = useQuery({
    queryKey: meKeys.retention(),
    queryFn: () => meApi.retention(),
    staleTime: 5 * 60_000,
  })

  const days = query.data
    ? kind === 'posts'
      ? query.data.posts_days
      : query.data.history_days
    : undefined
  if (days === undefined) return null

  return (
    <p className="mb-4 flex items-center gap-1.5 text-xs text-muted-foreground">
      <ShieldCheckIcon className="size-3.5 shrink-0" aria-hidden />
      {days === 0 ? t('forever') : t('days', { n: days })}
    </p>
  )
}
