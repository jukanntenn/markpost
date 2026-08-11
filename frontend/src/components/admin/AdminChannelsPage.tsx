'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { SendIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { SearchInput } from '@/components/ui/search-input'
import { Badge } from '@/components/ui/badge'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { PaginationControls } from '@/components/ui/pagination-controls'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
import type { AdminChannel } from '@/types/delivery'

// F.7 Admin 渠道管理页：搜索（名称/用户名）+ 开关/删除（确认文案如实告知
// 对用户投递的影响）+ 移动卡片。不见 webhook URL 配置（隐私，I.4）。
export function AdminChannelsPage() {
  const t = useTranslations('admin.channels')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()
  const queryClient = useQueryClient()

  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    search: string
  }>({ page: '1', search: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const debouncedSearch = useDebouncedValue(state.search, 300)

  const [toggleTarget, setToggleTarget] = useState<AdminChannel | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AdminChannel | null>(null)

  const query = useQuery({
    queryKey: adminKeys.channels.list(page),
    queryFn: () => adminApi.listChannels(page),
    staleTime: 30_000,
  })

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: adminKeys.channels.all() })
    queryClient.invalidateQueries({ queryKey: adminKeys.stats() })
  }

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      adminApi.setChannelEnabled(id, enabled),
    onSuccess: (_d, vars) => {
      setToggleTarget(null)
      invalidate()
      toastManager.add({
        type: 'success',
        title: vars.enabled ? t('enabledToast') : t('disabledToast'),
      })
    },
    onError: (err: Error) =>
      toastManager.add({ type: 'error', title: err.message }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => adminApi.deleteChannel(id),
    onSuccess: () => {
      setDeleteTarget(null)
      invalidate()
      toastManager.add({ type: 'success', title: t('deleted') })
    },
    onError: (err: Error) =>
      toastManager.add({ type: 'error', title: err.message }),
  })

  const channels = (query.data?.items ?? []).filter((c) => {
    if (!debouncedSearch) return true
    const q = debouncedSearch.toLowerCase()
    return (
      c.name.toLowerCase().includes(q) || c.username.toLowerCase().includes(q)
    )
  })
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0

  return (
    <div className="space-y-6">
      <PageHeading>{t('title')}</PageHeading>

      <div className="mb-4">
        <SearchInput
          placeholder={t('searchPlaceholder')}
          value={state.search}
          onChange={(v) => setState({ search: v })}
        />
      </div>

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={<Skeleton className="h-64 w-full" />}
        emptyWhen={channels.length === 0}
        empty={<EmptyState icon={SendIcon} title={t('empty')} />}
        onRetry={() => query.refetch()}
      >
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('name')}</TableHead>
                <TableHead>{t('kind')}</TableHead>
                <TableHead>{t('user')}</TableHead>
                <TableHead>{t('enabled')}</TableHead>
                <TableHead>{t('createdAt')}</TableHead>
                <TableHead className="w-32 text-right"> </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-medium">{c.name || '—'}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {c.kind}
                  </TableCell>
                  <TableCell>{c.username || '—'}</TableCell>
                  <TableCell>
                    <Badge variant={c.enabled ? 'success' : 'outline'}>
                      {c.enabled ? t('active') : t('inactive')}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {relativeTime(c.created_at, locale)}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Switch
                        size="sm"
                        checked={c.enabled}
                        onCheckedChange={(checked) =>
                          setToggleTarget({ ...c, enabled: checked })
                        }
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="text-danger hover:text-danger"
                        onClick={() => setDeleteTarget(c)}
                      >
                        {t('delete')}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <ul className="space-y-3 lg:hidden">
          {channels.map((c) => (
            <li key={c.id} className="rounded-lg border bg-card p-4">
              <div className="flex items-center justify-between gap-3">
                <span className="truncate font-semibold">{c.name || '—'}</span>
                <Badge variant={c.enabled ? 'success' : 'outline'}>
                  {c.enabled ? t('active') : t('inactive')}
                </Badge>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                {c.username || '—'} · {relativeTime(c.created_at, locale)}
              </p>
              <div className="mt-2 flex items-center justify-end gap-2">
                <Switch
                  size="sm"
                  checked={c.enabled}
                  onCheckedChange={(checked) =>
                    setToggleTarget({ ...c, enabled: checked })
                  }
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="text-danger hover:text-danger"
                  onClick={() => setDeleteTarget(c)}
                >
                  {t('delete')}
                </Button>
              </div>
            </li>
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

      <AlertDialog
        open={toggleTarget !== null}
        onOpenChange={(open) => !open && setToggleTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {toggleTarget?.enabled ? t('enableTitle') : t('disableTitle')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {toggleTarget?.enabled
                ? t('enableConfirm', {
                    name: toggleTarget?.name ?? '',
                    user: toggleTarget?.username ?? '',
                  })
                : t('disableConfirm', {
                    name: toggleTarget?.name ?? '',
                    user: toggleTarget?.username ?? '',
                  })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                toggleTarget &&
                toggleMutation.mutate({
                  id: toggleTarget.id,
                  enabled: toggleTarget.enabled,
                })
              }
            >
              {tCommon('confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteConfirm', { name: deleteTarget?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="danger"
              disabled={deleteMutation.isPending}
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.id)
              }
            >
              {t('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default AdminChannelsPage
