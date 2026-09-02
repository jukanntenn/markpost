'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { HistoryIcon, RefreshCwIcon } from 'lucide-react'
import { deliveryApi, deliveryKeys } from '@/lib/api'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { DEFAULT_PAGE_SIZE } from '@/lib/constants'
import { relativeTime } from '@/utils/relative-time'
import { formatToLocalTime } from '@/utils/time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { RetentionHint } from '@/components/me/RetentionHint'
import { PageHeading } from '@/components/ui/page-heading'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Badge, type BadgeVariant } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select } from '@base-ui/react/select'
import { ChevronDownIcon, CheckIcon } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { PaginationControls } from '@/components/ui/pagination-controls'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

// B3.4/F.4 投递历史页（多入口聚合）：渠道 + 状态筛选（URL 同步）、
// 失败详情展开、分页增强、移动卡片。
export function DeliveryHistoryPage() {
  const t = useTranslations('delivery.history')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()

  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    channel: string
    status: string
  }>({ page: '1', channel: '', status: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const channelId = state.channel
    ? Number.parseInt(state.channel, 10)
    : undefined
  const status = state.status || 'all'

  const channelsQuery = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: () => deliveryApi.list(),
    staleTime: 60_000,
  })

  const query = useQuery({
    queryKey: deliveryKeys.history(page, DEFAULT_PAGE_SIZE, channelId, status),
    queryFn: () =>
      deliveryApi.listHistory(page, DEFAULT_PAGE_SIZE, channelId, status),
    staleTime: 60_000,
  })

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0
  const hasFilters = state.channel !== '' || state.status !== ''

  // 失败详情展开（B3.3：inline 展开，可多个同时展开）。
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const toggleExpand = (id: number) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const statusOptions = [
    { value: 'all', label: tCommon('all') },
    { value: 'delivered', label: t('status.delivered') },
    { value: 'failed', label: t('status.failed') },
    { value: 'expired', label: t('status.expired') },
  ]

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

      <RetentionHint kind="history" />

      {/* 筛选（URL 同步，切筛选重置 page=1） */}
      <div className="flex flex-wrap items-center gap-3">
        <FilterSelect
          label={t('filter.channel')}
          value={state.channel || 'all'}
          options={[
            { value: 'all', label: tCommon('all') },
            ...(channelsQuery.data?.items ?? []).map((c) => ({
              value: String(c.id),
              label: c.name,
            })),
          ]}
          onValueChange={(v) => setState({ channel: v === 'all' ? '' : v })}
        />
        <FilterSelect
          label={t('filter.status')}
          value={status}
          options={statusOptions}
          onValueChange={(v) => setState({ status: v })}
        />
        {hasFilters && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setState({ channel: '', status: '' })}
          >
            {tCommon('clear')}
          </Button>
        )}
      </div>

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={<HistorySkeleton />}
        emptyWhen={items.length === 0}
        empty={
          <EmptyState
            icon={HistoryIcon}
            title={hasFilters ? t('emptyFiltered') : t('empty')}
            description={hasFilters ? undefined : t('emptyHint')}
            action={
              hasFilters ? (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setState({ channel: '', status: '' })}
                >
                  {tCommon('clear')}
                </Button>
              ) : undefined
            }
          />
        }
        onRetry={() => query.refetch()}
      >
        {/* 桌面 ≥lg 表格 */}
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('colPost')}</TableHead>
                <TableHead>{t('colChannel')}</TableHead>
                <TableHead>{t('colStatus')}</TableHead>
                <TableHead>{t('colTime')}</TableHead>
                <TableHead>{t('colError')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => {
                const failed =
                  row.status === 'failed' || row.status === 'expired'
                const isOpen = expanded.has(row.id)
                return (
                  <TableRow
                    key={row.id}
                    className={cn(
                      'cursor-pointer',
                      failed && isOpen && 'bg-danger/5',
                    )}
                    onClick={() => failed && toggleExpand(row.id)}
                  >
                    <TableCell className="max-w-56">
                      <span className="block truncate">
                        {row.post_title ?? t('postDeleted')}
                      </span>
                      {isOpen && failed && (
                        <span className="mt-1 block whitespace-normal text-xs text-danger">
                          {row.last_error}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm">
                      {row.channel_name ?? t('channelDeleted')}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={row.status} />
                    </TableCell>
                    <TableCell
                      className="text-sm text-muted-foreground"
                      title={formatToLocalTime(row.created_at, { locale })}
                    >
                      {relativeTime(row.created_at, locale)}
                    </TableCell>
                    <TableCell className="text-right">
                      {failed && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="xs"
                          onClick={(e) => {
                            e.stopPropagation()
                            toggleExpand(row.id)
                          }}
                        >
                          {t('detail')}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>

        {/* 移动 <lg 卡片 */}
        <ul className="space-y-3 lg:hidden">
          {items.map((row) => {
            const failed = row.status === 'failed' || row.status === 'expired'
            const isOpen = expanded.has(row.id)
            return (
              <li key={row.id} className="rounded-lg border bg-card p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate font-semibold">
                      {row.post_title ?? t('postDeleted')}
                    </p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {row.channel_name ?? t('channelDeleted')} ·{' '}
                      {relativeTime(row.created_at, locale)}
                    </p>
                  </div>
                  <StatusBadge status={row.status} />
                </div>
                {failed && (
                  <>
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => toggleExpand(row.id)}
                    >
                      {isOpen ? '−' : '+'} {t('detail')}
                    </Button>
                    {isOpen && (
                      <p className="mt-1 whitespace-normal text-xs text-danger">
                        {row.last_error}
                      </p>
                    )}
                  </>
                )}
              </li>
            )
          })}
        </ul>

        <PaginationControls
          page={page}
          totalPages={totalPages}
          total={total}
          onPageChange={setPage}
          prevLabel={t('prev')}
          nextLabel={t('next')}
          totalLabel={(n) => t('total', { n })}
        />
      </ListState>
    </div>
  )
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

export function StatusBadge({ status }: { status: string }) {
  const t = useTranslations('delivery.history.status')
  const variant: BadgeVariant =
    status === 'delivered'
      ? 'success'
      : status === 'failed'
        ? 'danger'
        : status === 'expired'
          ? 'outline'
          : 'secondary'
  return <Badge variant={variant}>{t(status as 'delivered') ?? status}</Badge>
}

function HistorySkeleton() {
  return (
    <div className="space-y-3">
      <div className="hidden overflow-hidden rounded-lg border lg:block">
        <div className="divide-y">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 p-3">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-5 w-16" />
              <Skeleton className="h-4 w-16" />
            </div>
          ))}
        </div>
      </div>
      <div className="space-y-3 lg:hidden">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-20 w-full rounded-lg" />
        ))}
      </div>
    </div>
  )
}

export default DeliveryHistoryPage
