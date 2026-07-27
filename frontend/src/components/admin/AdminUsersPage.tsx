'use client'

import { useTranslations } from 'next-intl'
import { UserPlusIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useAdminTablePage } from '@/hooks/useAdminTablePage'
import { formatToLocalTime } from '@/utils/time'
import { Button } from '@/components/ui/button'
import { TableHead, TableRow, TableCell } from '@/components/ui/table'
import { AdminTablePage } from '@/components/admin/AdminTablePage'
import { PaginationControls } from '@/components/ui/pagination-controls'

export function AdminUsersPage() {
  const t = useTranslations('admin')

  const {
    items: users,
    pagination,
    onPageChange,
    ...queryState
  } = useAdminTablePage({
    queryKey: adminKeys.users.all(),
    queryFn: (page, limit) => adminApi.listUsers(page, limit),
    t,
  })

  return (
    <>
      <AdminTablePage
        title={t('users.title')}
        toolbar={
          <Button>
            <UserPlusIcon className="mr-2 size-4" />
            {t('users.addUser')}
          </Button>
        }
        {...queryState}
        emptyText={t('noUsers')}
        headers={
          <>
            <TableHead>{t('id')}</TableHead>
            <TableHead>{t('username')}</TableHead>
            <TableHead>{t('role')}</TableHead>
            <TableHead>{t('createdAt')}</TableHead>
          </>
        }
        colSpan={4}
        items={users}
        renderRow={(user) => (
          <TableRow key={user.id}>
            <TableCell>{user.id}</TableCell>
            <TableCell>{user.username}</TableCell>
            <TableCell>{user.role}</TableCell>
            <TableCell>{formatToLocalTime(user.created_at)}</TableCell>
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
    </>
  )
}

export default AdminUsersPage
