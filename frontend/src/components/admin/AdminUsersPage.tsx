'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  UserPlusIcon,
  ShieldIcon,
  UserMinusIcon,
  Trash2Icon,
} from 'lucide-react'
import { adminApi, adminKeys, invalidateKey } from '@/lib/api'
import { mutationOptions } from '@/lib/mutation-helpers'
import { toast } from '@/stores/toast'
import { useAdminTablePage } from '@/hooks/useAdminTablePage'
import { formatToLocalTime } from '@/utils/time'
import { Button } from '@/components/ui/button'
import { TableHead, TableRow, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { AdminTablePage } from '@/components/admin/AdminTablePage'
import { AdminUserDialog } from '@/components/admin/AdminUserDialog'
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

type UserAction =
  | { type: 'role'; userId: number; username: string; newRole: string }
  | { type: 'password'; userId: number; username: string }
  | { type: 'active'; userId: number; username: string; active: boolean }
  | { type: 'delete'; userId: number; username: string }

export function AdminUsersPage() {
  const t = useTranslations('admin')
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [action, setAction] = useState<UserAction | null>(null)

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

  const roleMutation = useMutation(
    mutationOptions({
      mutationFn: ({ id, role }: { id: number; role: string }) =>
        adminApi.setUserRole(id, role),
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.users.all())
        toast.success(t('users.roleChanged'))
        setAction(null)
      },
    }),
  )

  const activeMutation = useMutation(
    mutationOptions({
      mutationFn: ({ id, active }: { id: number; active: boolean }) =>
        adminApi.setUserActive(id, active),
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.users.all())
        toast.success(t('users.activeChanged'))
        setAction(null)
      },
    }),
  )

  const deleteMutation = useMutation(
    mutationOptions({
      mutationFn: (id: number) => adminApi.deleteUser(id),
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.users.all())
        invalidateKey(queryClient, adminKeys.stats())
        toast.success(t('users.deleted'))
        setAction(null)
      },
    }),
  )

  function handleAction() {
    if (!action) return
    switch (action.type) {
      case 'role':
        roleMutation.mutate({
          id: action.userId,
          role: action.newRole,
        })
        break
      case 'active':
        activeMutation.mutate({
          id: action.userId,
          active: action.active,
        })
        break
      case 'delete':
        deleteMutation.mutate(action.userId)
        break
    }
  }

  function getActionTitle() {
    if (!action) return ''
    switch (action.type) {
      case 'role':
        return t('users.changeRole')
      case 'password':
        return t('users.resetPassword')
      case 'active':
        return action.active ? t('users.enableUser') : t('users.disableUser')
      case 'delete':
        return t('users.deleteTitle')
    }
  }

  function getActionDescription() {
    if (!action) return ''
    switch (action.type) {
      case 'role':
        return t('users.changeRoleConfirm', {
          username: action.username,
          role: action.newRole,
        })
      case 'active':
        return action.active
          ? t('users.enableUserConfirm', { username: action.username })
          : t('users.disableUserConfirm', { username: action.username })
      case 'delete':
        return t('users.deleteConfirm', { username: action.username })
    }
  }

  return (
    <>
      <AdminTablePage
        title={t('users.title')}
        toolbar={
          <Button onClick={() => setDialogOpen(true)}>
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
            <TableHead className="w-32">{t('users.actions')}</TableHead>
          </>
        }
        colSpan={5}
        items={users}
        renderRow={(user) => (
          <TableRow key={user.id}>
            <TableCell>{user.id}</TableCell>
            <TableCell>{user.username}</TableCell>
            <TableCell>
              <Badge variant={user.role === 'admin' ? 'default' : 'secondary'}>
                {user.role === 'admin' ? t('roleAdmin') : t('roleUser')}
              </Badge>
            </TableCell>
            <TableCell>{formatToLocalTime(user.created_at)}</TableCell>
            <TableCell>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  title={t('users.changeRole')}
                  onClick={() =>
                    setAction({
                      type: 'role',
                      userId: user.id,
                      username: user.username,
                      newRole: user.role === 'admin' ? 'user' : 'admin',
                    })
                  }
                >
                  <ShieldIcon className="size-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  title={
                    user.is_active
                      ? t('users.disableUser')
                      : t('users.enableUser')
                  }
                  onClick={() =>
                    setAction({
                      type: 'active',
                      userId: user.id,
                      username: user.username,
                      active: !user.is_active,
                    })
                  }
                >
                  <UserMinusIcon
                    className={`size-4 ${!user.is_active ? 'text-muted-foreground' : ''}`}
                  />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  title={t('users.deleteTitle')}
                  onClick={() =>
                    setAction({
                      type: 'delete',
                      userId: user.id,
                      username: user.username,
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

      <AdminUserDialog open={dialogOpen} onOpenChange={setDialogOpen} />

      <AlertDialog
        open={action !== null && action.type !== 'password'}
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
            <AlertDialogCancel>{t('users.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleAction}
              className={
                action?.type === 'delete'
                  ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                  : ''
              }
            >
              {t('users.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

export default AdminUsersPage
