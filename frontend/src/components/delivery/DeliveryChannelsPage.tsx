'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PlusIcon } from 'lucide-react'

import { deliveryApi, deliveryKeys, invalidateKey } from '@/lib/api'
import { mutationOptions } from '@/lib/mutation-helpers'
import { toast } from '@/stores/toast'
import { useDeliveryChannels } from '@/hooks/useDeliveryChannels'
import { truncate } from '@/lib/utils'
import { formatToLocalTime } from '@/utils/time'

import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { QueryState } from '@/components/ui/query-state'
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

export function DeliveryChannelsPage() {
  const t = useTranslations('delivery')
  const { channels, isLoading, error } = useDeliveryChannels()
  const queryClient = useQueryClient()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<DeliveryChannel | null>(
    null,
  )

  const latestQuery = useQuery({
    queryKey: deliveryKeys.latest(),
    queryFn: deliveryApi.latestPerChannel,
    refetchOnWindowFocus: false,
  })

  const toggleMutation = useMutation(
    mutationOptions({
      mutationFn: ({ id, data }: { id: number; data: UpdateChannelPayload }) =>
        deliveryApi.update(id, data),
      onSuccess: (_data, vars) => {
        invalidateKey(queryClient, deliveryKeys.channels())
        invalidateKey(queryClient, deliveryKeys.latest())
        toast.success(
          vars.data.enabled
            ? t('channels.enabledToast')
            : t('channels.disabledToast'),
        )
      },
    }),
  )

  function openNew() {
    setEditingChannel(null)
    setDialogOpen(true)
  }

  function openEdit(channel: DeliveryChannel) {
    setEditingChannel(channel)
    setDialogOpen(true)
  }

  const latestByChannel = new Map<number, DeliveryHistoryItem>()
  for (const item of latestQuery.data?.items ?? []) {
    if (item.channel_id !== null) {
      latestByChannel.set(item.channel_id, item)
    }
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

      <QueryState
        isLoading={isLoading}
        error={error}
        loadingText={t('channels.loading')}
        errorText={t('channels.loadFailed')}
        loadingClassName="flex items-center justify-center gap-2 py-8"
      >
        {channels.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-16 text-center">
            <p className="text-sm text-muted-foreground">
              {t('channels.empty')}
            </p>
            <Button variant="outline" size="sm" onClick={openNew}>
              <PlusIcon className="mr-1 size-4" />
              {t('channels.add')}
            </Button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border">
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
                        <p className="truncate text-xs text-muted-foreground">
                          {truncate(
                            channel.configuration?.webhook_url ?? '',
                            40,
                          )}
                        </p>
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
        )}
      </QueryState>

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

function LatestDeliveryCell({
  latest,
  loading,
}: {
  latest: DeliveryHistoryItem | undefined
  loading: boolean
}) {
  const t = useTranslations('delivery')

  if (loading) {
    return <Spinner className="size-4" />
  }
  if (!latest) {
    return (
      <span className="text-sm text-muted-foreground">{t('latest.never')}</span>
    )
  }

  const failed = latest.status === 'failed' || latest.status === 'expired'
  const tooltip = failed && latest.last_error ? latest.last_error : undefined

  return (
    <div className="flex items-center gap-2" title={tooltip}>
      <Badge variant={failed ? 'destructive' : 'secondary'}>
        {t(`history.status_${latest.status}`)}
      </Badge>
      <span className="text-xs text-muted-foreground">
        {formatToLocalTime(latest.created_at, { includeSeconds: false })}
      </span>
    </div>
  )
}

export default DeliveryChannelsPage
