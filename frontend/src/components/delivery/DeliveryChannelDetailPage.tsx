'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { deliveryApi, deliveryKeys } from '@/lib/api'
import { relativeTime } from '@/utils/relative-time'
import { formatToLocalTime } from '@/utils/time'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Badge } from '@/components/ui/badge'
import { Button, buttonClass } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { PaginationControls } from '@/components/ui/pagination-controls'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ArrowLeftIcon } from 'lucide-react'
import { StatusBadge } from './DeliveryHistoryPage'
import { DeliveryChannelDialog } from './DeliveryChannelDialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { toastManager } from '@/stores/toast'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import type {
  DeliveryChannel,
  DeliveryHistoryItem,
  DeliveryHistoryResponse,
} from '@/types/delivery'
import { Menu } from '@/components/ui/menu'
import { MoreHorizontalIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'

// F.3 渠道详情页：标题区主操作（启用/编辑/测试/删除，移动端折叠到 ⋯ 菜单）
// + 配置 + 最近投递（渠道维度历史）。
export function DeliveryChannelDetailPage() {
  const t = useTranslations('delivery.channelDetail')
  const tCommon = useTranslations('common')
  const tDialog = useTranslations('delivery.dialog')
  const router = useRouter()
  const queryClient = useQueryClient()
  const searchParams = useSearchParams()
  const { copied, copy } = useCopyToClipboard(2000)

  const channelId = Number.parseInt(searchParams.get('id') ?? '', 10)

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [testState, setTestState] = useState<
    'idle' | 'pending' | 'success' | 'failed'
  >('idle')

  // H.6 状态边界：分页进 URL query（?id=X&page=N，可分享/刷新保持）。
  const { state: pageState, setPage } = useUrlQueryState({ page: '1' })
  const page = Number.parseInt(pageState.page ?? '1', 10) || 1

  const channelsQuery = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: () => deliveryApi.list(),
    staleTime: 60_000,
  })

  const channel = Number.isNaN(channelId)
    ? undefined
    : channelsQuery.data?.items.find((c) => c.id === channelId)

  const historyQuery = useQuery<DeliveryHistoryResponse>({
    queryKey: deliveryKeys.history(page, 10, channelId),
    queryFn: () => deliveryApi.listHistory(page, 10, channelId),
    staleTime: 60_000,
  })

  const testMutation = useMutation({
    mutationFn: (id: number) => deliveryApi.test(id),
    onMutate: () => setTestState('pending'),
    onSuccess: () => {
      setTestState('success')
      toastManager.add({ type: 'success', title: tDialog('testSuccess') })
      setTimeout(() => setTestState('idle'), 3000)
    },
    onError: () => {
      setTestState('failed')
      toastManager.add({ type: 'error', title: tDialog('testFailed') })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deliveryApi.delete(id),
    onSuccess: () => {
      toastManager.add({ type: 'success', title: tDialog('deleted') })
      queryClient.invalidateQueries({ queryKey: deliveryKeys.channels() })
      router.push('/delivery/channels')
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  if (Number.isNaN(channelId)) {
    return <NotFound channelId={channelId} />
  }

  return (
    <div className="space-y-6">
      <Link
        href="/delivery/channels"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeftIcon className="size-4" />
        {t('back')}
      </Link>

      <ListState
        isLoading={channelsQuery.isLoading}
        error={channelsQuery.error}
        loadingSkeleton={<DetailSkeleton />}
        emptyWhen={!channel}
        empty={
          <EmptyState
            title={t('notFound')}
            action={
              <Link
                href="/delivery/channels"
                className={buttonClass('outline')}
              >
                {t('back')}
              </Link>
            }
          />
        }
        onRetry={() => channelsQuery.refetch()}
      >
        {channel && (
          <ChannelBody
            channel={channel}
            page={page}
            setPage={setPage}
            historyQuery={historyQuery}
            testState={testState}
            testPending={testMutation.isPending}
            onTest={() => testMutation.mutate(channel.id)}
            onEdit={() => setEditOpen(true)}
            onDelete={() => setDeleteOpen(true)}
            copied={copied}
            onCopyWebhook={() => copy(channel.configuration?.webhook_url ?? '')}
          />
        )}
      </ListState>

      {channel && editOpen && (
        <DeliveryChannelDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          editingChannel={channel}
        />
      )}

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {tDialog('deleteConfirm.title')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {tDialog('deleteConfirm.desc', { name: channel?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="danger"
              disabled={deleteMutation.isPending}
              onClick={() => channel && deleteMutation.mutate(channel.id)}
            >
              {tDialog('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function ChannelBody({
  channel,
  page,
  setPage,
  historyQuery,
  testState,
  testPending,
  onTest,
  onEdit,
  onDelete,
  copied,
  onCopyWebhook,
}: {
  channel: DeliveryChannel
  page: number
  setPage: (p: number) => void
  historyQuery: {
    data: DeliveryHistoryResponse | undefined
    isLoading: boolean
    error: Error | null
    refetch: () => void
  }
  testState: 'idle' | 'pending' | 'success' | 'failed'
  testPending: boolean
  onTest: () => void
  onEdit: () => void
  onDelete: () => void
  copied: boolean
  onCopyWebhook: () => void
}) {
  const t = useTranslations('delivery.channelDetail')
  const tDialog = useTranslations('delivery.dialog')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()

  const items: DeliveryHistoryItem[] = historyQuery.data?.items ?? []

  const HeaderActions = (
    <>
      <Badge variant={channel.enabled ? 'success' : 'outline'}>
        {channel.enabled ? t('enabled') : t('disabled')}
      </Badge>
      <Button
        type="button"
        size="sm"
        variant={testState === 'failed' ? 'danger' : 'outline'}
        disabled={testPending}
        onClick={onTest}
      >
        {testState === 'pending'
          ? tDialog('testing')
          : testState === 'success'
            ? tDialog('testSent')
            : testState === 'failed'
              ? tDialog('testRetry')
              : t('test')}
      </Button>
      <Button type="button" size="sm" onClick={onEdit}>
        {t('edit')}
      </Button>
      <Button type="button" size="sm" variant="danger" onClick={onDelete}>
        {t('delete')}
      </Button>
    </>
  )

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="min-w-0 flex-1 font-display text-headline font-bold tracking-tight">
          {channel.name || '—'}
        </h1>
        <div className="hidden items-center gap-2 md:flex">{HeaderActions}</div>
        {/* 移动端：主操作折叠到 ⋯ 菜单（F.3） */}
        <div className="md:hidden">
          <Menu.Root>
            <Menu.Trigger
              render={
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label={tCommon('search')}
                />
              }
            >
              <MoreHorizontalIcon className="size-4" />
            </Menu.Trigger>
            <Menu.Popup>
              <Menu.Item onClick={onTest}>{t('test')}</Menu.Item>
              <Menu.Item onClick={onEdit}>{t('edit')}</Menu.Item>
              <Menu.Item variant="danger" onClick={onDelete}>
                {t('delete')}
              </Menu.Item>
            </Menu.Popup>
          </Menu.Root>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('configTitle')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <ConfigRow label={t('webhookURL')}>
            <button
              type="button"
              onClick={onCopyWebhook}
              title={copied ? tCommon('copied') : tCommon('copy')}
              className="max-w-full truncate font-mono text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
              {channel.configuration?.webhook_url ?? '—'}
            </button>
          </ConfigRow>
          <ConfigRow label={t('cardLinkURL')}>
            <span className="max-w-full truncate font-mono text-xs text-muted-foreground">
              {channel.configuration?.card_link_url || '—'}
            </span>
          </ConfigRow>
          <ConfigRow label={t('keywords')}>
            <span className="text-muted-foreground">
              {channel.keywords || '—'}
            </span>
          </ConfigRow>
          <ConfigRow label={t('createdAt')}>
            <span className="text-muted-foreground">
              {formatToLocalTime(channel.created_at, {
                includeSeconds: false,
                locale,
              })}
            </span>
          </ConfigRow>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('historyTitle')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ListState
            isLoading={historyQuery.isLoading}
            error={historyQuery.error}
            loadingSkeleton={
              <div className="space-y-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            }
            emptyWhen={items.length === 0}
            empty={
              <EmptyState title={t('historyTitle')} description={t('empty')} />
            }
            onRetry={() => historyQuery.refetch()}
          >
            <ul className="divide-y">
              {items.map((row) => (
                <li
                  key={row.id}
                  className="flex items-center justify-between gap-3 py-2.5"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">
                      {row.post_title ?? '—'}
                    </p>
                    <p
                      className="mt-0.5 text-xs text-muted-foreground"
                      title={row.last_error || undefined}
                    >
                      {relativeTime(row.created_at, locale)}
                      {row.last_error && (
                        <span className="ml-2 text-danger">
                          {row.last_error}
                        </span>
                      )}
                    </p>
                  </div>
                  <StatusBadge status={row.status} />
                </li>
              ))}
            </ul>
            <PaginationControls
              page={page}
              totalPages={historyQuery.data?.total_pages ?? 0}
              total={historyQuery.data?.total ?? 0}
              onPageChange={setPage}
              prevLabel={tCommon('previous')}
              nextLabel={tCommon('next')}
              totalLabel={(n) => tCommon('total', { n })}
            />
          </ListState>
        </CardContent>
      </Card>
    </div>
  )
}

function ConfigRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="shrink-0 font-semibold">{label}</dt>
      <dd className="min-w-0 text-right">{children}</dd>
    </div>
  )
}

function NotFound({ channelId }: { channelId: number }) {
  const t = useTranslations('delivery.channelDetail')
  return (
    <EmptyState
      title={t('notFound')}
      description={Number.isNaN(channelId) ? '?id=' : `id=${channelId}`}
    />
  )
}

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-10 w-64" />
      <Skeleton className="h-48 w-full" />
      <Skeleton className="h-64 w-full" />
    </div>
  )
}

export default DeliveryChannelDetailPage
