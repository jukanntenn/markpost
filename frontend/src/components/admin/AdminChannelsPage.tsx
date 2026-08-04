'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { PowerIcon, Trash2Icon } from 'lucide-react'
import { adminApi, adminKeys, invalidateKey } from '@/lib/api'
import { mutationOptions } from '@/lib/mutation-helpers'
import { toast } from '@/stores/toast'
import { useAdminTablePage } from '@/hooks/useAdminTablePage'
import { formatToLocalTime } from '@/utils/time'
import { Button } from '@/components/ui/button'
import { TableHead, TableRow, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { AdminTablePage } from '@/components/admin/AdminTablePage'
import { PaginationControls } from '@/components/ui/pagination-controls'
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

type ChannelAction =
  | { type: 'enabled'; channelId: number; name: string; enabled: boolean }
  | { type: 'delete'; channelId: number; name: string }

export function AdminChannelsPage() {
  const t = useTranslations('admin')
  const queryClient = useQueryClient()
  const [action, setAction] = useState<ChannelAction | null>(null)

  const {
    items: channels,
    pagination,
    onPageChange,
    ...queryState
  } = useAdminTablePage({
    queryKey: adminKeys.channels.all(),
    queryFn: (page, limit) => adminApi.listChannels(page, limit),
    t,
  })

  const enabledMutation = useMutation(
    mutationOptions({
      mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
        adminApi.setChannelEnabled(id, enabled),
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.channels.all())
        toast.success(t('channels.enabledChanged'))
        setAction(null)
      },
    }),
  )

  const deleteMutation = useMutation(
    mutationOptions({
      mutationFn: (id: number) => adminApi.deleteChannel(id),
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.channels.all())
        invalidateKey(queryClient, adminKeys.stats())
        toast.success(t('channels.deleted'))
        setAction(null)
      },
    }),
  )

  function handleAction() {
    if (!action) return
    switch (action.type) {
      case 'enabled':
        enabledMutation.mutate({
          id: action.channelId,
          enabled: action.enabled,
        })
        break
      case 'delete':
        deleteMutation.mutate(action.channelId)
        break
    }
  }

  function getActionTitle() {
    if (!action) return ''
    switch (action.type) {
      case 'enabled':
        return action.enabled
          ? t('channels.enableTitle')
          : t('channels.disableTitle')
      case 'delete':
        return t('channels.deleteTitle')
    }
  }

  function getActionDescription() {
    if (!action) return ''
    switch (action.type) {
      case 'enabled':
        return action.enabled
          ? t('channels.enableConfirm', { name: action.name })
          : t('channels.disableConfirm', { name: action.name })
      case 'delete':
        return t('channels.deleteConfirm', { name: action.name })
    }
  }

  return (
    <>
      <AdminTablePage
        title={t('channels.title')}
        {...queryState}
        emptyText={t('channels.empty')}
        headers={
          <>
            <TableHead>{t('channels.id')}</TableHead>
            <TableHead>{t('channels.name')}</TableHead>
            <TableHead>{t('channels.kind')}</TableHead>
            <TableHead>{t('username')}</TableHead>
            <TableHead>{t('channels.enabled')}</TableHead>
            <TableHead>{t('createdAt')}</TableHead>
            <TableHead className="w-24">{t('channels.actions')}</TableHead>
          </>
        }
        colSpan={7}
        items={channels}
        renderRow={(channel) => (
          <TableRow key={channel.id}>
            <TableCell>{channel.id}</TableCell>
            <TableCell>{channel.name}</TableCell>
            <TableCell>
              <Badge variant="outline">{channel.kind}</Badge>
            </TableCell>
            <TableCell>{channel.username}</TableCell>
            <TableCell>
              <Badge variant={channel.enabled ? 'default' : 'secondary'}>
                {channel.enabled
                  ? t('channels.active')
                  : t('channels.inactive')}
              </Badge>
            </TableCell>
            <TableCell>{formatToLocalTime(channel.created_at)}</TableCell>
            <TableCell>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  title={
                    channel.enabled
                      ? t('channels.disableTitle')
                      : t('channels.enableTitle')
                  }
                  onClick={() =>
                    setAction({
                      type: 'enabled',
                      channelId: channel.id,
                      name: channel.name,
                      enabled: !channel.enabled,
                    })
                  }
                >
                  <PowerIcon
                    className={`size-4 ${!channel.enabled ? 'text-muted-foreground' : ''}`}
                  />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  title={t('channels.deleteTitle')}
                  onClick={() =>
                    setAction({
                      type: 'delete',
                      channelId: channel.id,
                      name: channel.name,
                    })
                  }
                >
                  <Trash2Icon className="size-4 text-destructive" />
                </Button>
              </div>
            </TableCell>
          </TableRow>
        )}
      />
      {pagination && pagination.total_pages > 1 && (
        <PaginationControls
          page={pagination.page}
          totalPages={pagination.total_pages}
          onPageChange={onPageChange}
          prevLabel={t('previous')}
          nextLabel={t('next')}
        />
      )}

      <AlertDialog
        open={action !== null}
        onOpenChange={(open) => !open && setAction(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{getActionTitle()}</AlertDialogTitle>
            <AlertDialogDescription>
              {getActionDescription()}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('channels.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleAction}
              className={
                action?.type === 'delete'
                  ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                  : ''
              }
            >
              {t('channels.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

export default AdminChannelsPage
