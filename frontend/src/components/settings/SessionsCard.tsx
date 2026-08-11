'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQuery } from '@tanstack/react-query'
import { authApi } from '@/lib/api'
import { sessionsKeys } from '@/lib/api/query-keys'
import { useAuthStore } from '@/stores/auth'
import { toastManager } from '@/stores/toast'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatToLocalTime } from '@/utils/time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
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

// I.12/F.2 用户透明：设置页安全区内的"我的会话"。
// 显示自己的 refresh tokens；可吊销单条会话 / 全部下线（保留当前 access）。
// F.2 裁决：会话区放安全卡片内（embedded 时去掉自身 Card 壳）。
export function SessionsCard({ embedded = false }: { embedded?: boolean }) {
  const t = useTranslations('settings.sessions')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()
  const user = useAuthStore((state) => state.user)

  const [revokeTarget, setRevokeTarget] = useState<number | null>(null)
  const [confirmLogoutAll, setConfirmLogoutAll] = useState(false)

  const query = useQuery({
    queryKey: sessionsKeys.current(),
    queryFn: () => authApi.listSessions(),
  })

  const invalidate = () => query.refetch()

  const revokeMutation = useMutation({
    mutationFn: (tokenId: number) => authApi.revokeSession(tokenId),
    onSuccess: () => {
      setRevokeTarget(null)
      toastManager.add({ type: 'success', title: t('revokedToast') })
      invalidate()
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  const logoutAllMutation = useMutation({
    mutationFn: () => authApi.revokeAllSessions(),
    onSuccess: () => {
      setConfirmLogoutAll(false)
      toastManager.add({ type: 'success', title: t('allSignedOut') })
      invalidate()
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  const sessions = query.data?.sessions ?? []

  // I.12/F.2：作为"安全"卡片内嵌区块（embedded）时去掉自身 Card 壳。
  const body = (
    <div className="space-y-3">
      {sessions.some((s) => !s.revoked && s.user_id === user?.id) && (
        <div className="flex justify-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={logoutAllMutation.isPending}
            onClick={() => setConfirmLogoutAll(true)}
          >
            {t('logoutAll')}
          </Button>
        </div>
      )}

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        }
        emptyWhen={sessions.length === 0}
        empty={<EmptyState title={t('empty')} />}
        onRetry={() => query.refetch()}
      >
        <ul className="divide-y">
          {sessions.map((s) => (
            <li
              key={s.id}
              className="flex items-center justify-between gap-3 py-2.5"
            >
              <div className="min-w-0 text-sm">
                <p className="flex items-center gap-2">
                  <span
                    className={`size-2 rounded-full ${s.revoked ? 'bg-muted-foreground' : 'bg-success'}`}
                    aria-hidden="true"
                  />
                  {s.revoked ? t('revoked') : t('active')}
                </p>
                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                  {t('issued')}:{' '}
                  {formatToLocalTime(s.created_at, {
                    includeSeconds: false,
                    locale,
                  })}{' '}
                  · {t('expires')}:{' '}
                  {formatToLocalTime(s.expires_at, {
                    includeSeconds: false,
                    locale,
                  })}
                </p>
              </div>
              {!s.revoked && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={revokeMutation.isPending}
                  onClick={() => setRevokeTarget(s.id)}
                >
                  {t('revoke')}
                </Button>
              )}
            </li>
          ))}
        </ul>
      </ListState>

      <AlertDialog
        open={revokeTarget !== null}
        onOpenChange={(open) => !open && setRevokeTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('revokeTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('revokeConfirm')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant="danger"
              disabled={revokeMutation.isPending}
              onClick={() =>
                revokeTarget !== null && revokeMutation.mutate(revokeTarget)
              }
            >
              {t('revoke')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirmLogoutAll} onOpenChange={setConfirmLogoutAll}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('logoutAll')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('logoutAllConfirm')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={logoutAllMutation.isPending}
              onClick={() => logoutAllMutation.mutate()}
            >
              {t('logoutAll')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )

  if (embedded) {
    return (
      <section aria-label={t('title')} className="mt-6">
        <h3 className="mb-2 text-sm font-semibold">{t('title')}</h3>
        {body}
      </section>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('title')}</CardTitle>
      </CardHeader>
      <CardContent>{body}</CardContent>
    </Card>
  )
}
