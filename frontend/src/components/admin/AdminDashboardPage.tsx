'use client'

import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { adminApi, adminKeys } from '@/lib/api'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ListState } from '@/components/ui/list-state'
import { Skeleton } from '@/components/ui/skeleton'
import { DeliveryTrendChart } from '@/components/dashboard/DeliveryTrendChart'
import { CircleCheckIcon, TriangleAlertIcon } from 'lucide-react'
import type { AuditLogItem } from '@/types/audit'
import { auditActionText } from '@/lib/audit-action-text'

// D2 Admin 态势感知仪表盘：需要关注（D2.1/D2.2）+ 最近管理操作（D2.3）
// + 存量指标环比（D2.4）+ 投递趋势（D2.5）。
export function AdminDashboardPage() {
  const t = useTranslations('admin.dashboard')
  const { locale } = useLocaleContext()

  const statsQuery = useQuery({
    queryKey: adminKeys.stats(),
    queryFn: () => adminApi.getStats(),
    staleTime: 60_000,
  })
  const lockedQuery = useQuery({
    queryKey: adminKeys.lockedChannels(),
    queryFn: () => adminApi.lockedChannels(),
    staleTime: 60_000,
  })
  const trendQuery = useQuery({
    queryKey: adminKeys.trend(7),
    queryFn: () => adminApi.deliveryStats(7),
    staleTime: 60_000,
  })
  const auditQuery = useQuery({
    queryKey: adminKeys.audit.list(1, {}),
    queryFn: () => adminApi.listAuditLogs(1, {}, 5),
    staleTime: 30_000,
  })

  const counts = statsQuery.data?.counts
  const locked = lockedQuery.data?.items ?? []
  const bannedCount = counts?.banned_users ?? 0

  // D2.2 需要关注项：投递失败渠道 + 被封禁用户（>0 显示）。
  const attentionItems: { id: string; text: string; href: string }[] = []
  for (const ch of locked) {
    attentionItems.push({
      id: `channel-${ch.channel_id}`,
      text: t('needsAttention.channelFailing', {
        name: ch.channel_name || `#${ch.channel_id}`,
        n: ch.fails,
      }),
      href: `/admin/delivery/history?channel_id=${ch.channel_id}&status=failed`,
    })
  }
  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between gap-3">
        <h1 className="font-display text-headline font-bold tracking-tight">
          {t('title')}
        </h1>
        <Badge variant="success">
          <CircleCheckIcon className="size-3" />
          {t('systemOk')}
        </Badge>
      </header>

      <div className="grid gap-6 xl:grid-cols-[1fr_340px]">
        <div className="space-y-6">
          {/* D2.2 需要关注 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                {t('needsAttention.title')}
                {attentionItems.length > 0 && (
                  <Badge variant="warning" className="ml-2">
                    {attentionItems.length}
                  </Badge>
                )}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ListState
                isLoading={lockedQuery.isLoading}
                error={lockedQuery.error}
                loadingSkeleton={
                  <div className="space-y-2">
                    {Array.from({ length: 2 }).map((_, i) => (
                      <Skeleton key={i} className="h-10 w-full" />
                    ))}
                  </div>
                }
                onRetry={() => lockedQuery.refetch()}
                errorTitle={t('needsAttention.detectError')}
              >
                {attentionItems.length === 0 ? (
                  <div className="flex items-center gap-2 text-sm text-success">
                    <CircleCheckIcon className="size-4" />
                    <span className="font-semibold">
                      {t('needsAttention.allGood')}
                    </span>
                    <span className="text-muted-foreground">
                      {t('needsAttention.noIssue')}
                    </span>
                  </div>
                ) : (
                  <ul className="space-y-2">
                    {attentionItems.map((item) => (
                      <li
                        key={item.id}
                        className="flex items-center justify-between gap-3 rounded-md border px-3 py-2.5"
                      >
                        <span className="flex min-w-0 items-center gap-2 text-sm">
                          <TriangleAlertIcon className="size-4 shrink-0 text-warning" />
                          <span className="truncate">{item.text}</span>
                        </span>
                        <Link
                          href={item.href}
                          className="shrink-0 text-sm font-medium text-primary hover:underline"
                        >
                          {t('needsAttention.viewChannel')}
                        </Link>
                      </li>
                    ))}
                    {bannedCount > 0 && (
                      <li className="flex items-center justify-between gap-3 rounded-md border px-3 py-2.5">
                        <span className="flex min-w-0 items-center gap-2 text-sm">
                          <TriangleAlertIcon className="size-4 shrink-0 text-warning" />
                          <span className="truncate">
                            {t('needsAttention.usersBanned', {
                              n: bannedCount,
                            })}
                          </span>
                        </span>
                        <Link
                          href="/admin/users?status=disabled"
                          className="shrink-0 text-sm font-medium text-primary hover:underline"
                        >
                          {t('needsAttention.viewUser')}
                        </Link>
                      </li>
                    )}
                  </ul>
                )}
              </ListState>
            </CardContent>
          </Card>

          {/* D2.3 最近管理操作（审计流） */}
          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <CardTitle className="text-base">
                {t('recentActions.title')}
              </CardTitle>
              <Link
                href="/admin/audit-logs"
                className="text-sm text-primary hover:underline"
              >
                {t('recentActions.viewAll')}
              </Link>
            </CardHeader>
            <CardContent>
              <ListState
                isLoading={auditQuery.isLoading}
                error={auditQuery.error}
                loadingSkeleton={<Skeleton className="h-32 w-full" />}
                onRetry={() => auditQuery.refetch()}
              >
                <AuditFeed
                  rows={auditQuery.data?.items ?? []}
                  locale={locale}
                />
              </ListState>
            </CardContent>
          </Card>
        </div>

        <aside className="space-y-6">
          {/* D2.4 存量指标 + 环比 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t('stats.users')}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-2 gap-4">
              <StatCell
                label={t('stats.users')}
                value={counts?.users}
                delta={counts?.users_week_delta}
                loading={statsQuery.isLoading}
                deltaLabel={t('stats.weekDelta')}
              />
              <StatCell
                label={t('stats.posts')}
                value={counts?.posts}
                delta={counts?.posts_week_delta}
                loading={statsQuery.isLoading}
                deltaLabel={t('stats.weekDelta')}
              />
              <StatCell
                label={t('stats.channels')}
                value={counts?.channels}
                loading={statsQuery.isLoading}
              />
              <StatCell
                label={t('stats.history')}
                value={counts?.history}
                delta={counts?.history_week_delta}
                loading={statsQuery.isLoading}
                deltaLabel={t('stats.weekDelta')}
              />
            </CardContent>
          </Card>

          {/* D2.5 投递趋势 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t('trend.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              <ListState
                isLoading={trendQuery.isLoading}
                error={trendQuery.error}
                loadingSkeleton={<Skeleton className="h-56 w-full" />}
                onRetry={() => trendQuery.refetch()}
              >
                <DeliveryTrendChart data={trendQuery.data?.trend ?? []} />
              </ListState>
            </CardContent>
          </Card>
        </aside>
      </div>
    </div>
  )
}

function StatCell({
  label,
  value,
  delta,
  loading,
  deltaLabel,
}: {
  label: string
  value?: number
  delta?: number
  loading: boolean
  deltaLabel?: string
}) {
  if (loading) return <Skeleton className="h-16 w-full" />
  return (
    <div className="rounded-lg border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 font-display text-section font-bold">
        {value?.toLocaleString() ?? '—'}
      </p>
      {delta != null && deltaLabel && (
        <p
          className={
            delta > 0
              ? 'text-xs text-success'
              : delta < 0
                ? 'text-xs text-danger'
                : 'text-xs text-muted-foreground'
          }
        >
          {delta > 0 ? deltaLabel.replace('{n}', String(delta)) : '—'}
        </p>
      )}
    </div>
  )
}

// D2.3 审计流：{时间} {操作者} {动作(过去式)} {对象}，点击行跳审计页。
function AuditAction({ row }: { row: AuditLogItem }) {
  const tAction = useTranslations('admin.audit.action')
  const { key, values } = auditActionText(row)
  return <span>{tAction(key as never, values as never)}</span>
}

function AuditFeed({ rows, locale }: { rows: AuditLogItem[]; locale: string }) {
  const t = useTranslations('admin.dashboard.recentActions')
  if (rows.length === 0) {
    return <p className="text-sm text-muted-foreground">{t('viewAll')}</p>
  }
  return (
    <ul className="space-y-2">
      {rows.map((row) => (
        <li key={row.id}>
          <Link
            href="/admin/audit-logs"
            className="block rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent"
          >
            <span className="text-xs text-muted-foreground">
              {relativeTime(row.created_at, locale)}
            </span>{' '}
            <span className="font-medium">@{row.actor_username || '?'}</span>{' '}
            <AuditAction row={row} />
          </Link>
        </li>
      ))}
    </ul>
  )
}

export default AdminDashboardPage
