'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { HistoryIcon } from 'lucide-react'
import { adminApi, adminKeys, deliveryKeys } from '@/lib/api'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { relativeTime } from '@/utils/relative-time'
import { formatToLocalTime } from '@/utils/time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { PaginationControls } from '@/components/ui/pagination-controls'
import { Select } from '@base-ui/react/select'
import { ChevronDownIcon, CheckIcon } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { StatusBadge } from '../delivery/DeliveryHistoryPage'
import type { DeliveryHistoryItem } from '@/types/delivery'

// F.8 Admin 投递历史页：用户/渠道/状态筛选（I.10 白名单，无时间筛选）+ 失败详情
// 展开 + 移动卡片。
export function AdminDeliveryHistoryPage() {
  const t = useTranslations('admin.history')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()

  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    user: string
    channel: string
    status: string
    category: string
  }>({ page: '1', user: '', channel: '', status: '', category: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const userId = state.user ? Number.parseInt(state.user, 10) : undefined
  const channelId = state.channel
    ? Number.parseInt(state.channel, 10)
    : undefined
  const status = state.status || 'all'
  const category = state.category || 'all'

  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const toggle = (id: number) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const channelsQuery = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: () => adminApi.listChannels(1, 100),
    staleTime: 60_000,
  })

  const filter = {
    ...(userId ? { user_id: userId } : {}),
    ...(channelId ? { channel_id: channelId } : {}),
    ...(status !== 'all' ? { status } : {}),
    ...(category !== 'all' ? { error_category: category } : {}),
  }
  const query = useQuery({
    queryKey: adminKeys.history.list(page, filter),
    queryFn: () => adminApi.listDeliveryHistory(page, filter),
    staleTime: 30_000,
  })

  const items: DeliveryHistoryItem[] = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0
  const hasFilters =
    state.user !== '' ||
    state.channel !== '' ||
    status !== 'all' ||
    category !== 'all'

  return (
    <div className="space-y-6">
      <PageHeading>{t('title')}</PageHeading>

      <div className="flex flex-wrap items-center gap-3">
        <FilterSelect
          label={t('filter.user')}
          value={state.user || 'all'}
          options={[
            { value: 'all', label: tCommon('all') },
            { value: state.user || 'all', label: state.user || tCommon('all') },
          ]}
          onValueChange={(v) => setState({ user: v === 'all' ? '' : v })}
        />
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
          options={[
            { value: 'all', label: tCommon('all') },
            { value: 'delivered', label: t('status.delivered') },
            { value: 'failed', label: t('status.failed') },
            { value: 'expired', label: t('status.expired') },
          ]}
          onValueChange={(v) => setState({ status: v })}
        />
        <FilterSelect
          label={t('filter.errorCategory')}
          value={category}
          options={[
            { value: 'all', label: tCommon('all') },
            { value: 'card_rejected', label: t('errorCategory.card_rejected') },
            {
              value: 'upstream_client_error',
              label: t('errorCategory.upstream_client_error'),
            },
            {
              value: 'upstream_server_error',
              label: t('errorCategory.upstream_server_error'),
            },
            {
              value: 'upstream_business_error',
              label: t('errorCategory.upstream_business_error'),
            },
            { value: 'network', label: t('errorCategory.network') },
            { value: 'internal', label: t('errorCategory.internal') },
          ]}
          onValueChange={(v) => setState({ category: v })}
        />
        {hasFilters && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() =>
              setState({ user: '', channel: '', status: '', category: '' })
            }
          >
            {tCommon('clear')}
          </Button>
        )}
      </div>

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={<Skeleton className="h-64 w-full" />}
        emptyWhen={items.length === 0}
        empty={
          <EmptyState
            icon={HistoryIcon}
            title={hasFilters ? t('emptyFiltered') : t('empty')}
            action={
              hasFilters ? (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() =>
                    setState({
                      user: '',
                      channel: '',
                      status: '',
                      category: '',
                    })
                  }
                >
                  {tCommon('clear')}
                </Button>
              ) : undefined
            }
          />
        }
        onRetry={() => query.refetch()}
      >
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('colPost')}</TableHead>
                <TableHead>{t('colChannel')}</TableHead>
                <TableHead>{t('colUser')}</TableHead>
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
                    className={failed ? 'cursor-pointer' : undefined}
                    onClick={() => failed && toggle(row.id)}
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
                    <TableCell>
                      {row.channel_name ?? t('channelDeleted')}
                    </TableCell>
                    <TableCell>{row.username ?? t('userDeleted')}</TableCell>
                    <TableCell>
                      <div className="flex flex-col items-start gap-1">
                        <StatusBadge status={row.status} />
                        <CategoryTag category={row.error_category} />
                      </div>
                    </TableCell>
                    <TableCell
                      className="text-muted-foreground"
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
                            toggle(row.id)
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
                      {row.username ?? t('userDeleted')} ·{' '}
                      {relativeTime(row.created_at, locale)}
                    </p>
                  </div>
                  <StatusBadge status={row.status} />
                </div>
                <CategoryTag category={row.error_category} />
                {failed && (
                  <>
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => toggle(row.id)}
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
          prevLabel={tCommon('previous')}
          nextLabel={tCommon('next')}
          totalLabel={(n) => tCommon('total', { n })}
        />
      </ListState>
    </div>
  )
}

function CategoryTag({ category }: { category: string }) {
  const t = useTranslations('admin.history')
  if (!category) return null
  return (
    <span className="rounded border border-input bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">
      {t(`errorCategory.${category}`)}
    </span>
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

export default AdminDeliveryHistoryPage
