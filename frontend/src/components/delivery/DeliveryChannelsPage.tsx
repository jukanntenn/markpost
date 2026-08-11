'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PlusIcon, SendIcon } from 'lucide-react'

import { deliveryApi, deliveryKeys } from '@/lib/api'
import { toastManager } from '@/stores/toast'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge, type BadgeVariant } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type {
  DeliveryChannel,
  DeliveryHistoryItem,
  UpdateChannelPayload,
} from '@/types/delivery'
import { DeliveryChannelDialog } from './DeliveryChannelDialog'

// F.4 渠道列表页：桌面表格（lg）+ 移动卡片（同一数据渲染分离，B3.2）。
// 渠道开关 = 乐观更新（H.3/I.1：onError 回滚 + 失败 toast）。
export function DeliveryChannelsPage() {
  const t = useTranslations('delivery')
  const { locale } = useLocaleContext()
  const queryClient = useQueryClient()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<DeliveryChannel | null>(
    null,
  )

  const channelsQuery = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: () => deliveryApi.list(),
    staleTime: 60_000,
  })
  const latestQuery = useQuery({
    queryKey: deliveryKeys.latest(),
    queryFn: () => deliveryApi.latestPerChannel(),
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })

  const channels = channelsQuery.data?.items ?? []

  const toggleMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateChannelPayload }) =>
      deliveryApi.update(id, data),
    // I.1 乐观更新（渠道开关）：立即翻转 UI，失败回滚 + toast。
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: deliveryKeys.channels() })
      const prev = queryClient.getQueryData(deliveryKeys.channels())
      queryClient.setQueryData(
        deliveryKeys.channels(),
        (old: { items: DeliveryChannel[] } | undefined) =>
          old
            ? {
                ...old,
                items: old.items.map((c) =>
                  c.id === id
                    ? { ...c, enabled: data.enabled ?? c.enabled }
                    : c,
                ),
              }
            : old,
      )
      return { prev }
    },
    onError: (err, vars, ctx) => {
      if (ctx?.prev) {
        queryClient.setQueryData(deliveryKeys.channels(), ctx.prev)
      }
      toastManager.add({ type: 'error', title: err.message })
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: deliveryKeys.channels() })
      queryClient.invalidateQueries({ queryKey: deliveryKeys.latest() })
    },
    onSuccess: (_data, vars) => {
      toastManager.add({
        type: 'success',
        title: vars.data.enabled
          ? t('channels.enabledToast')
          : t('channels.disabledToast'),
      })
    },
  })

  const latestByChannel = new Map<number, DeliveryHistoryItem>()
  for (const item of latestQuery.data?.items ?? []) {
    if (item.channel_id !== null) {
      latestByChannel.set(item.channel_id, item)
    }
  }

  const openNew = () => {
    setEditingChannel(null)
    setDialogOpen(true)
  }
  const openEdit = (channel: DeliveryChannel) => {
    setEditingChannel(channel)
    setDialogOpen(true)
  }

  return (
    <div className="space-y-6">
      <PageHeading
        actions={
          <Button onClick={openNew}>
            <PlusIcon className="mr-1 size-4" />
            {t('channels.add')}
          </Button>
        }
      >
        {t('channels.title')}
      </PageHeading>

      <ListState
        isLoading={channelsQuery.isLoading}
        error={channelsQuery.error}
        loadingSkeleton={<ChannelsSkeleton />}
        emptyWhen={channels.length === 0}
        empty={
          <EmptyState
            icon={SendIcon}
            title={t('channels.empty')}
            description={t('channels.emptyCta')}
            action={
              <Button onClick={openNew}>
                <PlusIcon className="mr-1 size-4" />
                {t('channels.add')}
              </Button>
            }
          />
        }
        onRetry={() => channelsQuery.refetch()}
      >
        {/* 桌面 ≥lg：完整表格 */}
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('channels.colName')}</TableHead>
                <TableHead>{t('channels.colType')}</TableHead>
                <TableHead className="w-[80px]">
                  {t('channels.colEnabled')}
                </TableHead>
                <TableHead>{t('channels.colKeywords')}</TableHead>
                <TableHead>{t('channels.colLatest')}</TableHead>
                <TableHead className="w-[120px] text-right">
                  {t('channels.colActions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.map((channel) => {
                const latest = latestByChannel.get(channel.id)
                return (
                  <TableRow key={channel.id}>
                    <TableCell>
                      <Link
                        href={`/delivery/channel?id=${channel.id}`}
                        className="font-medium hover:underline"
                      >
                        {channel.name || t('channels.unnamed')}
                      </Link>
                    </TableCell>
                    <TableCell className="text-sm">{channel.kind}</TableCell>
                    <TableCell>
                      <Switch
                        size="sm"
                        checked={channel.enabled}
                        disabled={toggleMutation.isPending}
                        onCheckedChange={(checked) =>
                          toggleMutation.mutate({
                            id: channel.id,
                            data: { enabled: checked },
                          })
                        }
                      />
                    </TableCell>
                    <TableCell className="max-w-[180px] truncate text-sm text-muted-foreground">
                      {channel.keywords || '—'}
                    </TableCell>
                    <TableCell>
                      <LatestDeliveryCell
                        latest={latest}
                        loading={latestQuery.isLoading}
                        locale={locale}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openEdit(channel)}
                      >
                        {t('channels.edit')}
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>

        {/* 移动 <lg：卡片列表（信息不删减，B3.2） */}
        <ul className="space-y-3 lg:hidden">
          {channels.map((channel) => {
            const latest = latestByChannel.get(channel.id)
            const failed =
              latest != null &&
              (latest.status === 'failed' || latest.status === 'expired')
            return (
              <li key={channel.id} className="rounded-lg border bg-card p-4">
                <div className="flex items-center justify-between gap-3">
                  <Link
                    href={`/delivery/channel?id=${channel.id}`}
                    className="min-w-0 truncate font-semibold hover:underline"
                  >
                    {channel.name || t('channels.unnamed')}
                  </Link>
                  <Switch
                    size="sm"
                    checked={channel.enabled}
                    disabled={toggleMutation.isPending}
                    onCheckedChange={(checked) =>
                      toggleMutation.mutate({
                        id: channel.id,
                        data: { enabled: checked },
                      })
                    }
                  />
                </div>
                <p className="mt-1 truncate text-xs text-muted-foreground">
                  {channel.kind}
                  {channel.keywords ? ` · ${channel.keywords}` : ''}
                  {latest
                    ? ` · ${failed ? t('history.status.failed') : '✓'} ${
                        latest.created_at
                          ? relativeTime(latest.created_at, locale)
                          : ''
                      }`
                    : ` · ${t('channels.latest.never')}`}
                </p>
                <div className="mt-2 flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">
                    {failed && latest?.last_error
                      ? latest.last_error
                      : (channel.configuration?.webhook_url ?? '')}
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => openEdit(channel)}
                  >
                    {t('channels.edit')}
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      </ListState>

      {dialogOpen && (
        <DeliveryChannelDialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          editingChannel={editingChannel}
        />
      )}
    </div>
  )
}

function ChannelsSkeleton() {
  return (
    <div className="space-y-3">
      <div className="hidden overflow-hidden rounded-lg border lg:block">
        <div className="divide-y">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 p-3">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-5 w-8" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-20" />
            </div>
          ))}
        </div>
      </div>
      <div className="space-y-3 lg:hidden">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-24 w-full rounded-lg" />
        ))}
      </div>
    </div>
  )
}

function LatestDeliveryCell({
  latest,
  loading,
  locale,
}: {
  latest: DeliveryHistoryItem | undefined
  loading: boolean
  locale: string
}) {
  const t = useTranslations('delivery')

  if (loading) {
    return <Skeleton className="h-4 w-20" />
  }
  if (!latest) {
    return (
      <span className="text-sm text-muted-foreground">
        {t('channels.latest.never')}
      </span>
    )
  }

  const failed = latest.status === 'failed' || latest.status === 'expired'
  const variant: BadgeVariant = failed ? 'danger' : 'success'
  const statusKey =
    latest.status === 'pending'
      ? 'history.status.pending'
      : `history.status.${latest.status}`

  return (
    <div
      className="flex items-center gap-2"
      title={failed ? latest.last_error : undefined}
    >
      <Badge variant={variant}>{t(statusKey)}</Badge>
      <span className="text-xs text-muted-foreground">
        {latest.created_at ? relativeTime(latest.created_at, locale) : ''}
      </span>
    </div>
  )
}

export default DeliveryChannelsPage
