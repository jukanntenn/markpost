'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { RefreshCwIcon, ScrollTextIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { auditActionText } from '@/lib/audit-action-text'
import { formatToLocalTime } from '@/utils/time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { PaginationControls } from '@/components/ui/pagination-controls'
import { Select } from '@base-ui/react/select'
import { ChevronDownIcon, CheckIcon } from 'lucide-react'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { toastManager } from '@/stores/toast'
import type { AuditLogItem, AuditFilters } from '@/types/audit'

// D4 审计日志页：4 维筛选（操作者/动作/对象类型/时间，URL 同步，debounce 300ms）、
// 行展开元数据（已知键 i18n）、IP 脱敏、分页增强、移动卡片。
export function AdminAuditLogsPage() {
  const t = useTranslations('admin.audit')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()
  const { copied, copy: copyText } = useCopyToClipboard(2000)

  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    actor: string
    action: string
    target_type: string
    time: string
  }>({ page: '1', actor: '', action: '', target_type: '', time: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const debouncedActor = useDebouncedValue(state.actor, 300)

  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const toggle = (id: number) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  // 时间范围起点在查询时计算（Date.now 属查询副作用，不参与渲染纯度）。
  const sinceMs = useMemo(() => {
    if (!state.time || state.time === 'all') return 0
    return state.time === 'today'
      ? 86_400_000
      : state.time === '7d'
        ? 7 * 86_400_000
        : 30 * 86_400_000
  }, [state.time])

  const filters: AuditFilters = useMemo(() => {
    const f: AuditFilters = {}
    if (debouncedActor)
      f.actor_id = Number.parseInt(debouncedActor, 10) || undefined
    if (state.action && state.action !== 'all') f.action = state.action
    if (state.target_type && state.target_type !== 'all')
      f.target_type = state.target_type
    return f
  }, [debouncedActor, state.action, state.target_type])

  const query = useQuery({
    queryKey: adminKeys.audit.list(page, { ...filters, since: sinceMs }),
    queryFn: () =>
      adminApi.listAuditLogs(page, {
        ...filters,
        since:
          sinceMs > 0
            ? new Date(Date.now() - sinceMs).toISOString()
            : undefined,
      }),
    staleTime: 30_000,
  })

  const rows = query.data?.items ?? []
  const facets = query.data?.facets ?? {}
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0

  const hasFilters =
    debouncedActor !== '' ||
    (state.action && state.action !== 'all') ||
    (state.target_type && state.target_type !== 'all') ||
    (state.time && state.time !== 'all')

  const activeFilterCount = [
    debouncedActor !== '',
    state.action && state.action !== 'all',
    state.target_type && state.target_type !== 'all',
    state.time && state.time !== 'all',
  ].filter(Boolean).length

  const actionOptions = [
    { value: 'all', label: tCommon('all') },
    ...Object.keys(facets)
      .sort()
      .map((a) => ({ value: a, label: `${actionLabel(a)} (${facets[a]})` })),
  ]

  const targetOptions = [
    { value: 'all', label: tCommon('all') },
    { value: 'user', label: 'user' },
    { value: 'post', label: 'post' },
    { value: 'channel', label: 'channel' },
    { value: 'session', label: 'session' },
  ]

  const timeOptions = [
    { value: 'all', label: tCommon('all') },
    { value: 'today', label: t('filter.today') },
    { value: '7d', label: t('filter.days7') },
    { value: '30d', label: t('filter.days30') },
  ]

  const clearAll = () =>
    setState({ actor: '', action: '', target_type: '', time: '' })

  return (
    <div className="space-y-6">
      <PageHeading
        actions={
          <button
            type="button"
            onClick={() => query.refetch()}
            aria-label={tCommon('refresh')}
            className="flex size-11 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
          >
            <RefreshCwIcon className="size-4" />
          </button>
        }
      >
        {t('title')}
      </PageHeading>

      {/* 筛选器（D4.3：URL 同步；操作者搜索 debounce 300ms；动作带计数） */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">
            {t('filter.actor')}
          </span>
          <Input
            value={state.actor}
            onChange={(e) => setState({ actor: e.target.value })}
            placeholder="id / username"
            className="h-10 w-40"
            aria-label={t('filter.actor')}
          />
        </div>
        <FilterSelect
          label={t('filter.action')}
          value={state.action || 'all'}
          options={actionOptions}
          onValueChange={(v) => setState({ action: v })}
        />
        <FilterSelect
          label={t('filter.targetType')}
          value={state.target_type || 'all'}
          options={targetOptions}
          onValueChange={(v) => setState({ target_type: v })}
        />
        <FilterSelect
          label={t('filter.timeRange')}
          value={state.time || 'all'}
          options={timeOptions}
          onValueChange={(v) => setState({ time: v })}
        />
        {hasFilters && (
          <Button type="button" variant="ghost" size="sm" onClick={clearAll}>
            {tCommon('clear')}
          </Button>
        )}
      </div>

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={<AuditSkeleton />}
        emptyWhen={rows.length === 0}
        empty={
          <EmptyState
            icon={ScrollTextIcon}
            title={hasFilters ? t('emptyFiltered') : t('empty')}
            action={
              hasFilters ? (
                <Button type="button" variant="outline" onClick={clearAll}>
                  {tCommon('clear')}
                </Button>
              ) : undefined
            }
          />
        }
        onRetry={() => query.refetch()}
      >
        {/* 桌面 ≥lg */}
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <table className="w-full caption-bottom text-sm">
            <thead>
              <tr className="border-b">
                <th
                  scope="col"
                  className="h-10 px-3 text-left font-semibold whitespace-nowrap"
                >
                  {t('colTime')}
                </th>
                <th
                  scope="col"
                  className="h-10 px-3 text-left font-semibold whitespace-nowrap"
                >
                  {t('colActor')}
                </th>
                <th
                  scope="col"
                  className="h-10 px-3 text-left font-semibold whitespace-nowrap"
                >
                  {t('colAction')}
                </th>
                <th
                  scope="col"
                  className="h-10 px-3 text-left font-semibold whitespace-nowrap"
                >
                  {t('colTarget')}
                </th>
                <th
                  scope="col"
                  className="h-10 px-3 text-left font-semibold whitespace-nowrap"
                >
                  {t('colIp')}
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <AuditRowDesktop
                  key={row.id}
                  row={row}
                  expanded={expanded.has(row.id)}
                  onToggle={() => toggle(row.id)}
                  copied={copied}
                  onCopy={async (text: string) => {
                    await copyText(text)
                  }}
                  locale={locale}
                />
              ))}
            </tbody>
          </table>
        </div>

        {/* 移动 <lg */}
        <ul className="space-y-3 lg:hidden">
          {rows.map((row) => (
            <AuditRowMobile
              key={row.id}
              row={row}
              expanded={expanded.has(row.id)}
              onToggle={() => toggle(row.id)}
            />
          ))}
        </ul>

        <PaginationControls
          page={page}
          totalPages={totalPages}
          total={total}
          onPageChange={setPage}
          prevLabel={tCommon('previous')}
          nextLabel={tCommon('next')}
          totalLabel={(n) => tCommon('total', { n })}
        />
      </ListState>

      {/* 移动端筛选折叠按钮（D4.6）——筛选已在顶部响应式换行，这里标注激活数 */}
      <p className="sr-only" aria-live="polite">
        {activeFilterCount > 0
          ? `${activeFilterCount} ${t('filter.action')}`
          : ''}
      </p>
    </div>
  )
}

function actionLabel(action: string) {
  // 计数下拉的 label 用动作 key 本身（i18n 由行渲染）。
  return action
}

function AuditRowDesktop({
  row,
  expanded,
  onToggle,
  copied,
  onCopy,
  locale,
}: {
  row: AuditLogItem
  expanded: boolean
  onToggle: () => void
  copied: boolean
  onCopy: (text: string) => Promise<void>
  locale: string
}) {
  const t = useTranslations('admin.audit')

  return (
    <>
      <tr
        onClick={onToggle}
        aria-expanded={expanded}
        className="cursor-pointer border-b transition-colors hover:bg-muted"
      >
        <td className="px-3 py-2.5 whitespace-nowrap text-muted-foreground">
          <span title={formatToLocalTime(row.created_at, { locale })}>
            {formatToLocalTime(row.created_at, {
              includeSeconds: false,
              locale,
            })}
          </span>
        </td>
        <td className="px-3 py-2.5 whitespace-nowrap">
          @{row.actor_username || row.actor_id}
        </td>
        <td className="px-3 py-2.5">
          <ActionText row={row} />
        </td>
        <td className="px-3 py-2.5 whitespace-nowrap">
          {/* D4.2 对象列：类型 badge + ID（user/post 可下钻详情） */}
          <span className="inline-flex items-center gap-1.5">
            <span className="rounded-sm bg-muted px-1.5 py-0.5 text-xs font-medium text-muted-foreground">
              {row.target_type}
            </span>
            {targetHref(row.target_type, row.target_id) ? (
              <Link
                href={targetHref(row.target_type, row.target_id) as string}
                onClick={(e) => e.stopPropagation()}
                className="text-primary underline-offset-4 hover:underline"
              >
                {row.target_id}
              </Link>
            ) : (
              <span className="text-muted-foreground">{row.target_id}</span>
            )}
          </span>
        </td>
        <td className="px-3 py-2.5 whitespace-nowrap text-muted-foreground">
          <span title={row.ip || ''}>{maskIP(row.ip)}</span>
          <span className="ml-2 text-xs">{expanded ? '▾' : '▸'}</span>
        </td>
      </tr>
      {expanded && (
        <tr className="border-b bg-muted/40">
          <td colSpan={5} className="px-3 py-3">
            <div className="space-y-1 text-xs">
              <p className="font-semibold">{t('metadata')}</p>
              <MetadataList row={row} />
              <div className="mt-2 flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  onClick={() => {
                    onCopy(JSON.stringify(row.metadata ?? {}, null, 2))
                    if (!copied)
                      toastManager.add({
                        type: 'success',
                        title: t('copyJson'),
                      })
                  }}
                >
                  {t('copyJson')}
                </Button>
                <span className="text-muted-foreground" title={row.ip || ''}>
                  {t('showFullIp')}: {row.ip || '—'}
                </span>
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

function AuditRowMobile({
  row,
  expanded,
  onToggle,
}: {
  row: AuditLogItem
  expanded: boolean
  onToggle: () => void
}) {
  return (
    <li className="rounded-lg border bg-card p-4">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="w-full text-left"
      >
        <p className="font-semibold">
          <ActionText row={row} />
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          @{row.actor_username || row.actor_id} ·{' '}
          {formatToLocalTime(row.created_at, { includeSeconds: false })}
        </p>
      </button>
      {expanded && (
        <p className="mt-2 whitespace-normal text-xs text-muted-foreground">
          {JSON.stringify(row.metadata ?? {})}
        </p>
      )}
    </li>
  )
}

function ActionText({ row }: { row: AuditLogItem }) {
  const tAction = useTranslations('admin.audit.action')
  const { key, values } = auditActionText(row)
  return <span>{tAction(key as never, values as never)}</span>
}

// targetHref returns the drill-down URL for an audit target (D4.2). user →
// admin user detail (?id=), post → public render page. channel/session have no
// admin-facing detail page, so they render as plain text.
function targetHref(targetType: string, targetID: string): string | null {
  if (targetType === 'user')
    return `/admin/users?id=${encodeURIComponent(targetID)}`
  if (targetType === 'post') return `/${encodeURIComponent(targetID)}`
  return null
}

function MetadataList({ row }: { row: AuditLogItem }) {
  const tMeta = useTranslations('admin.audit.meta')
  const meta = row.metadata
  if (!meta || Object.keys(meta).length === 0) {
    return <p className="text-muted-foreground">—</p>
  }
  const items: { label: string; value: string }[] = []
  for (const [k, v] of Object.entries(meta)) {
    if (k === 'from' || k === 'to') {
      items.push({
        label: tMeta('transition'),
        value: `${meta.from ?? '—'} → ${meta.to ?? '—'}`,
      })
    } else if (k === 'reason') {
      items.push({ label: tMeta('reason'), value: String(v) })
    } else if (k === 'via') {
      items.push({ label: tMeta('via'), value: String(v) })
    } else {
      items.push({ label: k, value: String(v) })
    }
  }
  return (
    <ul className="space-y-0.5">
      {items.map((it, i) => (
        <li key={i}>
          {it.label}: <span className="text-muted-foreground">{it.value}</span>
        </li>
      ))}
    </ul>
  )
}

// D4.2 IP 脱敏：1.2.x.x，hover 完整。
function maskIP(ip: string): string {
  if (!ip) return '—'
  const parts = ip.split('.')
  if (parts.length !== 4) return ip
  return `${parts[0]}.${parts[1]}.x.x`
}

function FilterSelect({
  label,
  value,
  options,
  onValueChange,
}: {
  label: string
  value: string
  options: { value: string; label: string }[]
  onValueChange: (value: string) => void
}) {
  const labelMap = new Map(options.map((o) => [o.value, o.label]))
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-muted-foreground">{label}</span>
      <Select.Root
        items={options}
        value={value}
        onValueChange={(v, details) => {
          if (details.reason === 'none') return
          onValueChange(v ?? '')
        }}
      >
        <Select.Trigger className="flex h-10 min-w-36 items-center justify-between gap-2 rounded-md border border-input bg-card px-3 py-2 text-sm select-none focus-visible:border-primary focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring">
          <Select.Value className="data-[placeholder]:text-muted-foreground" />
          <Select.Icon>
            <ChevronDownIcon className="size-4 text-muted-foreground" />
          </Select.Icon>
        </Select.Trigger>
        <Select.Portal>
          <Select.Positioner sideOffset={4}>
            <Select.Popup className="z-[100] min-w-44 overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-lg outline-none">
              <Select.List>
                {options.map((o) => (
                  <Select.Item
                    key={o.value}
                    value={o.value}
                    className="flex cursor-default items-center justify-between gap-2 rounded-sm px-2 py-1.5 text-sm outline-none select-none data-[highlighted]:bg-accent"
                  >
                    <Select.ItemText>{labelMap.get(o.value)}</Select.ItemText>
                    <Select.ItemIndicator className="flex items-center">
                      <CheckIcon className="size-4 text-primary" />
                    </Select.ItemIndicator>
                  </Select.Item>
                ))}
              </Select.List>
            </Select.Popup>
          </Select.Positioner>
        </Select.Portal>
      </Select.Root>
    </div>
  )
}

function AuditSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 5 }).map((_, i) => (
        <Skeleton key={i} className="h-11 w-full" />
      ))}
    </div>
  )
}

export default AdminAuditLogsPage
