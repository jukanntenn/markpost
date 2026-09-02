'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { ArrowLeftIcon } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminApi, adminKeys } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { toastManager } from '@/stores/toast'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { formatToLocalTime } from '@/utils/time'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VipBadge } from '@/components/ui/vip-badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from './AdminUsersPage'
import { UserGovernanceDialogs, type PendingAction } from './UserGovernance'
import { RetentionDialog, RetentionPolicyText } from './RetentionDialog'
import { auditActionText } from '@/lib/audit-action-text'

// D3.2 用户详情页（/admin/users?id=，静态导出约束）：资料 + 治理操作
// + 会话 + 操作历史。
export function AdminUserDetailPage() {
  const t = useTranslations('admin')
  const tCommon = useTranslations('common')
  const tUsers = useTranslations('admin.users')
  const tSessions = useTranslations('settings.sessions')
  const { locale } = useLocaleContext()
  const queryClient = useQueryClient()
  const searchParams = useSearchParams()
  const me = useAuthStore((state) => state.user)
  const { copied, copy } = useCopyToClipboard(2000)

  const userId = Number.parseInt(searchParams.get('id') ?? '', 10)
  const [action, setAction] = useState<PendingAction | null>(null)
  const [retentionOpen, setRetentionOpen] = useState(false)

  const userQuery = useQuery({
    queryKey: adminKeys.users.detail(userId),
    queryFn: () => adminApi.getUser(userId),
    staleTime: 30_000,
  })
  const sessionsQuery = useQuery({
    queryKey: adminKeys.sessions(userId),
    queryFn: () => adminApi.listSessions(userId),
    staleTime: 30_000,
  })
  const historyQuery = useQuery({
    queryKey: adminKeys.audit.list(1, {
      target_id: String(userId),
      target_type: 'user',
    }),
    queryFn: () =>
      adminApi.listAuditLogs(
        1,
        { target_id: String(userId), target_type: 'user' },
        10,
      ),
    staleTime: 30_000,
  })

  const user = userQuery.data
  const sessions = sessionsQuery.data?.sessions ?? []
  const history = historyQuery.data?.items ?? []

  const logoutAllMutation = useMutation({
    mutationFn: () => adminApi.revokeAllSessions(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminKeys.sessions(userId) })
      queryClient.invalidateQueries({ queryKey: adminKeys.users.all() })
      toastManager.add({
        type: 'success',
        title: tUsers('detail.sessionsRevokedToast'),
      })
    },
    onError: (err: Error) =>
      toastManager.add({ type: 'error', title: err.message }),
  })

  const revokeMutation = useMutation({
    mutationFn: (tokenId: number) => adminApi.revokeSession(tokenId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminKeys.sessions(userId) })
      toastManager.add({
        type: 'success',
        title: tUsers('detail.sessionsRevokedToast'),
      })
    },
    onError: (err: Error) =>
      toastManager.add({ type: 'error', title: err.message }),
  })

  if (Number.isNaN(userId)) {
    return (
      <EmptyState
        title={tUsers('detail.back')}
        action={
          <Link
            href="/admin/users"
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeftIcon className="size-4" />
            {tUsers('detail.back')}
          </Link>
        }
      />
    )
  }

  const isSelf = me?.id === userId

  return (
    <div className="space-y-6">
      <Link
        href="/admin/users"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeftIcon className="size-4" />
        {tUsers('detail.back')}
      </Link>

      <ListState
        isLoading={userQuery.isLoading}
        error={userQuery.error}
        loadingSkeleton={
          <div className="space-y-6">
            <Skeleton className="h-10 w-64" />
            <div className="grid gap-6 lg:grid-cols-[1fr_260px]">
              <Skeleton className="h-64 w-full" />
              <Skeleton className="h-40 w-full" />
            </div>
          </div>
        }
        emptyWhen={!user}
        empty={
          <EmptyState
            title={t('empty')}
            description={t('users.detail.back')}
            action={
              <Link
                href="/admin/users"
                className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
              >
                <ArrowLeftIcon className="size-4" />
                {tUsers('detail.back')}
              </Link>
            }
          />
        }
        onRetry={() => userQuery.refetch()}
      >
        {user && (
          <>
            <header className="flex flex-wrap items-center gap-3">
              <h1 className="min-w-0 flex-1 font-display text-headline font-bold tracking-tight">
                {user.username}
              </h1>
              <Badge variant={user.role === 'admin' ? 'default' : 'outline'}>
                {user.role === 'admin' ? t('roleAdmin') : t('roleUser')}
              </Badge>
              {user.vip && <VipBadge />}
              <StatusBadge active={user.is_active} />
            </header>
            <p className="text-sm text-muted-foreground">
              @{user.username} · {t('createdAt')}{' '}
              {formatToLocalTime(user.created_at, {
                includeSeconds: false,
                locale,
              })}
            </p>

            <div className="grid gap-6 lg:grid-cols-[1fr_280px]">
              {/* 资料区 */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">
                    {tUsers('detail.profile')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <ProfileRow label={t('id')} value={String(user.id)} />
                  <ProfileRow label={t('username')} value={user.username} />
                  <ProfileRow
                    label={tUsers('detail.fullName')}
                    value={user.name || '—'}
                  />
                  <ProfileRow
                    label={t('email')}
                    value={user.email || tUsers('detail.emailUnbound')}
                  >
                    {/* D3.2 邮箱验证标记 */}
                    {user.email && (
                      <Badge
                        variant={user.is_email_verified ? 'success' : 'outline'}
                      >
                        {user.is_email_verified
                          ? tUsers('detail.emailVerified')
                          : tUsers('detail.emailNotVerified')}
                      </Badge>
                    )}
                  </ProfileRow>
                  <ProfileRow
                    label={t('githubId')}
                    value={
                      user.github_id
                        ? tUsers('detail.githubLinked', { id: user.github_id })
                        : tUsers('detail.githubUnbound')
                    }
                  />
                  <ProfileRow
                    label={t('role')}
                    value={
                      user.role === 'admin' ? t('roleAdmin') : t('roleUser')
                    }
                  />
                  <ProfileRow
                    label={t('status.normal')}
                    value={
                      user.is_active ? t('status.normal') : t('status.disabled')
                    }
                  />
                  <ProfileRow label={tUsers('detail.vip')} value="">
                    {user.vip ? <VipBadge /> : <span>—</span>}
                  </ProfileRow>
                  <ProfileRow label={tUsers('retention.column')}>
                    <span className="flex items-center gap-2">
                      <RetentionPolicyText days={user.retention_days} />
                      <button
                        type="button"
                        className="text-xs text-primary hover:underline"
                        onClick={() => setRetentionOpen(true)}
                        data-testid="detail-set-retention"
                      >
                        {tUsers('retention.setLink')}
                      </button>
                    </span>
                  </ProfileRow>
                  <ProfileRow
                    label={t('lastLoginAt')}
                    value={
                      user.last_login_at
                        ? formatToLocalTime(user.last_login_at, {
                            includeSeconds: false,
                            locale,
                          })
                        : '—'
                    }
                  />
                  <ProfileRow label={tUsers('detail.postKey')}>
                    <span className="flex items-center gap-2">
                      <code className="font-mono text-xs">
                        {'•'.repeat(Math.min(user.post_key?.length ?? 0, 16))}
                      </code>
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        onClick={() => {
                          copy(user.post_key)
                          toastManager.add({
                            type: 'success',
                            title: tUsers('detail.postKeyCopied'),
                          })
                        }}
                      >
                        {copied
                          ? tUsers('detail.postKeyCopied')
                          : tCommon('copy')}
                      </Button>
                    </span>
                  </ProfileRow>
                </CardContent>
              </Card>

              {/* 操作区（D3.3 六个治理操作入口） */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">
                    {tUsers('detail.actions')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setAction({ kind: 'reset' })}
                  >
                    {tUsers('resetPassword')}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={isSelf}
                    title={isSelf ? tUsers('role.selfDemote') : undefined}
                    onClick={() =>
                      setAction({
                        kind: 'role',
                        to: user.role === 'admin' ? 'user' : 'admin',
                      })
                    }
                  >
                    {user.role === 'admin'
                      ? tUsers('role.toUser.title')
                      : tUsers('role.toAdmin.title')}
                  </Button>
                  <Button
                    type="button"
                    variant={user.is_active ? 'danger' : 'outline'}
                    disabled={isSelf}
                    title={isSelf ? tUsers('role.selfDemote') : undefined}
                    onClick={() =>
                      setAction({ kind: user.is_active ? 'disable' : 'enable' })
                    }
                  >
                    {user.is_active
                      ? tUsers('disableUser')
                      : tUsers('enableUser')}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setAction({ kind: 'forceLogout' })}
                  >
                    {tUsers('forceLogout')}
                  </Button>
                  <Button
                    type="button"
                    variant="danger"
                    disabled={isSelf}
                    onClick={() => setAction({ kind: 'delete' })}
                  >
                    {tUsers('deleteTitle')}
                  </Button>
                </CardContent>
              </Card>
            </div>

            {/* 会话区 */}
            <Card>
              <CardHeader className="flex-row items-center justify-between">
                <CardTitle className="text-base">
                  {tUsers('detail.sessions')}
                </CardTitle>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={logoutAllMutation.isPending}
                  onClick={() => logoutAllMutation.mutate()}
                >
                  {tUsers('detail.logoutAll')}
                </Button>
              </CardHeader>
              <CardContent>
                <ListState
                  isLoading={sessionsQuery.isLoading}
                  error={sessionsQuery.error}
                  loadingSkeleton={<Skeleton className="h-24 w-full" />}
                  emptyWhen={sessions.length === 0}
                  empty={
                    <EmptyState
                      title={tUsers('detail.sessionsEmpty')}
                      className="border-0 py-8"
                    />
                  }
                  onRetry={() => sessionsQuery.refetch()}
                >
                  <ul className="divide-y">
                    {sessions.map((s) => (
                      <li
                        key={s.id}
                        className="flex items-center justify-between gap-3 py-2.5 text-sm"
                      >
                        <span>
                          <span className="font-medium">
                            {formatToLocalTime(s.created_at, {
                              includeSeconds: false,
                              locale,
                            })}
                          </span>
                          <span className="ml-2 text-muted-foreground">
                            {tUsers('detail.sessions')} ·{' '}
                            {formatToLocalTime(s.expires_at, {
                              includeSeconds: false,
                              locale,
                            })}
                          </span>
                        </span>
                        <span className="flex items-center gap-2">
                          <Badge variant={s.revoked ? 'outline' : 'success'}>
                            {s.revoked
                              ? tSessions('revoked')
                              : tSessions('active')}
                          </Badge>
                          {!s.revoked && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="xs"
                              disabled={revokeMutation.isPending}
                              onClick={() => revokeMutation.mutate(s.id)}
                            >
                              {tSessions('revoke')}
                            </Button>
                          )}
                        </span>
                      </li>
                    ))}
                  </ul>
                </ListState>
              </CardContent>
            </Card>

            {/* 操作历史 */}
            <Card>
              <CardHeader className="flex-row items-center justify-between">
                <CardTitle className="text-base">
                  {tUsers('detail.history')}
                </CardTitle>
                <Link
                  href={`/admin/audit-logs?target_type=user&target_id=${user.id}`}
                  className="text-sm text-primary hover:underline"
                >
                  {tUsers('detail.historyViewAll')}
                </Link>
              </CardHeader>
              <CardContent>
                <ListState
                  isLoading={historyQuery.isLoading}
                  error={historyQuery.error}
                  loadingSkeleton={<Skeleton className="h-24 w-full" />}
                  emptyWhen={history.length === 0}
                  empty={
                    <EmptyState
                      title={tUsers('detail.historyEmpty')}
                      className="border-0 py-8"
                    />
                  }
                  onRetry={() => historyQuery.refetch()}
                >
                  <ul className="space-y-1">
                    {history.map((row) => (
                      <li
                        key={row.id}
                        className="rounded-md px-2 py-1.5 text-sm"
                      >
                        <span className="text-xs text-muted-foreground">
                          {relativeTime(row.created_at, locale)}
                        </span>{' '}
                        <span className="font-medium">
                          @{row.actor_username || '?'}
                        </span>{' '}
                        <AuditText row={row} />
                      </li>
                    ))}
                  </ul>
                </ListState>
              </CardContent>
            </Card>
          </>
        )}
      </ListState>

      {user && (
        <UserGovernanceDialogs
          user={user}
          action={action}
          onClose={() => setAction(null)}
        />
      )}

      {retentionOpen && user && (
        <RetentionDialog
          target={{
            kind: 'user',
            id: user.id,
            username: user.username,
            current: user.retention_days,
          }}
          open={retentionOpen}
          onOpenChange={setRetentionOpen}
        />
      )}
    </div>
  )
}

function ProfileRow({
  label,
  value,
  children,
}: {
  label: string
  value?: string
  children?: React.ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="shrink-0 font-semibold">{label}</dt>
      <dd className="min-w-0 text-right text-muted-foreground">
        {children ?? <span className="break-all">{value}</span>}
      </dd>
    </div>
  )
}

function AuditText({
  row,
}: {
  row: {
    action: string
    target_id: string
    metadata: Record<string, unknown> | null
  }
}) {
  const t = useTranslations('admin.audit.action')
  const { key, values } = auditActionText(row as never)
  return <span>{t(key as never, values as never)}</span>
}

export default AdminUserDetailPage
