'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { useRouter } from 'next/navigation'
import { PlusIcon, UsersIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { SearchInput } from '@/components/ui/search-input'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
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
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { AdminUserDialog } from './AdminUserDialog'
import {
  UserActionsMenu,
  UserGovernanceDialogs,
  type PendingAction,
} from './UserGovernance'
import type { AdminUser } from '@/types/users'

// D3.1 用户列表：搜索（debounce 300ms）+ 状态 badge + 详情入口 + ⋮ 快捷操作。
export function AdminUsersPage() {
  const t = useTranslations('admin')
  const tUsers = useTranslations('admin.users')
  const { locale } = useLocaleContext()
  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    search: string
  }>({ page: '1', search: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const debouncedSearch = useDebouncedValue(state.search, 300)

  const [createOpen, setCreateOpen] = useState(false)
  const [action, setAction] = useState<PendingAction | null>(null)
  const [actionUser, setActionUser] = useState<AdminUser | null>(null)
  const router = useRouter()

  const query = useQuery({
    queryKey: adminKeys.users.list(page, debouncedSearch),
    queryFn: () => adminApi.listUsers(page, debouncedSearch),
    staleTime: 30_000,
  })

  const users = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0

  return (
    <div className="space-y-6">
      <PageHeading
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <PlusIcon className="mr-1 size-4" />
            {tUsers('addUser')}
          </Button>
        }
      >
        {tUsers('title')}
      </PageHeading>

      <div className="mb-4">
        <SearchInput
          placeholder={t('searchUserPlaceholder')}
          value={state.search}
          onChange={(v) => setState({ search: v })}
        />
      </div>

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        }
        emptyWhen={users.length === 0}
        empty={
          <EmptyState
            icon={UsersIcon}
            title={t('empty')}
            action={
              <Button onClick={() => setCreateOpen(true)}>
                {tUsers('addUser')}
              </Button>
            }
          />
        }
        onRetry={() => query.refetch()}
      >
        {/* 桌面表格 */}
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('id')}</TableHead>
                <TableHead>{t('username')}</TableHead>
                <TableHead>{t('role')}</TableHead>
                <TableHead>{t('status.normal')}</TableHead>
                <TableHead>{t('createdAt')}</TableHead>
                <TableHead className="w-32 text-right">
                  {tUsers('actions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-sm text-muted-foreground">
                    {u.id}
                  </TableCell>
                  <TableCell className="font-medium">
                    <a
                      href={`/admin/users?id=${u.id}`}
                      className="hover:underline"
                    >
                      {u.username}
                    </a>
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.role === 'admin' ? 'default' : 'outline'}>
                      {u.role === 'admin' ? t('roleAdmin') : t('roleUser')}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge active={u.is_active} />
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {relativeTime(u.created_at, locale)}
                  </TableCell>
                  <TableCell className="text-right">
                    <UserActionsMenu
                      user={u}
                      onOpenDetail={(target) =>
                        router.push(`/admin/users?id=${target.id}`)
                      }
                      onAction={(a) => {
                        setActionUser(u)
                        setAction(a)
                      }}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        {/* 移动卡片 */}
        <ul className="space-y-3 lg:hidden">
          {users.map((u) => (
            <li key={u.id} className="rounded-lg border bg-card p-4">
              <div className="flex items-center justify-between gap-3">
                <a
                  href={`/admin/users?id=${u.id}`}
                  className="min-w-0 truncate font-semibold hover:underline"
                >
                  {u.username}
                </a>
                <Badge variant={u.role === 'admin' ? 'default' : 'outline'}>
                  {u.role === 'admin' ? t('roleAdmin') : t('roleUser')}
                </Badge>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                {t('id')} {u.id} · {t('createdAt')}{' '}
                {relativeTime(u.created_at, locale)}
              </p>
              <div className="mt-2 flex items-center justify-between">
                <StatusBadge active={u.is_active} />
                <a
                  href={`/admin/users?id=${u.id}`}
                  className="text-sm text-primary hover:underline"
                >
                  {tUsers('viewDetail')}
                </a>
              </div>
            </li>
          ))}
        </ul>

        <PaginationControls
          page={page}
          totalPages={totalPages}
          total={total}
          onPageChange={setPage}
          prevLabel={t('previous')}
          nextLabel={t('next')}
          totalLabel={(n) => t('total', { n })}
        />
      </ListState>

      {createOpen && (
        <AdminUserDialog open={createOpen} onOpenChange={setCreateOpen} />
      )}

      {actionUser && (
        <UserGovernanceDialogs
          user={actionUser}
          action={action}
          onClose={() => setAction(null)}
        />
      )}
    </div>
  )
}

export function StatusBadge({ active }: { active: boolean }) {
  const t = useTranslations('admin.status')
  return (
    <Badge variant={active ? 'success' : 'danger'}>
      <span
        className={`mr-1 size-1.5 rounded-full ${active ? 'bg-success-foreground' : 'bg-danger-foreground'}`}
      />
      {active ? t('normal') : t('disabled')}
    </Badge>
  )
}

export default AdminUsersPage
