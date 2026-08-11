'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { MoreHorizontalIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { toastManager } from '@/stores/toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Menu } from '@/components/ui/menu'
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import type { AdminUser } from '@/types/users'

// D3.3 六个治理操作的统一交互范式：点击 → AlertDialog（告知后果）→ 确认
// → mutation → 反馈 + invalidate + 关闭。防自操作（I.4/K.7）由后端兜底 +
// 前端隐藏（selfDemote 场景）。
type PendingAction =
  | { kind: 'role'; to: 'admin' | 'user' }
  | { kind: 'disable' }
  | { kind: 'enable' }
  | { kind: 'forceLogout' }
  | { kind: 'reset' }
  | { kind: 'delete' }

export type { PendingAction }

// 列表行 ⋮ 菜单：详情 + 快捷治理操作。
export function UserActionsMenu({
  user,
  onOpenDetail,
  onAction,
}: {
  user: AdminUser
  onOpenDetail: (user: AdminUser) => void
  onAction: (action: PendingAction) => void
}) {
  const t = useTranslations('admin.users')
  const tRole = useTranslations('admin')
  const me = useAuthStore((state) => state.user)

  const isSelf = me?.id === user.id

  return (
    <Menu.Root>
      <Menu.Trigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t('moreActions')}
          />
        }
      >
        <MoreHorizontalIcon className="size-4" />
      </Menu.Trigger>
      <Menu.Popup>
        <Menu.Item onClick={() => onOpenDetail(user)}>
          {t('viewDetail')}
        </Menu.Item>
        <Menu.Separator />
        {/* I.4 防自操作：对自己隐藏 切换角色/启停/删除（不渲染，D3.3 双保险） */}
        {!isSelf && (
          <Menu.Item
            onClick={() =>
              onAction({
                kind: 'role',
                to: user.role === 'admin' ? 'user' : 'admin',
              })
            }
          >
            {user.role === 'admin'
              ? t('changeRole') + ' → ' + tRole('roleUser')
              : t('changeRole')}
          </Menu.Item>
        )}
        <Menu.Item onClick={() => onAction({ kind: 'reset' })}>
          {t('resetPassword')}
        </Menu.Item>
        {!isSelf &&
          (user.is_active ? (
            <Menu.Item onClick={() => onAction({ kind: 'disable' })}>
              {t('disableUser')}
            </Menu.Item>
          ) : (
            <Menu.Item onClick={() => onAction({ kind: 'enable' })}>
              {t('enableUser')}
            </Menu.Item>
          ))}
        <Menu.Item onClick={() => onAction({ kind: 'forceLogout' })}>
          {t('forceLogout')}
        </Menu.Item>
        {!isSelf && (
          <>
            <Menu.Separator />
            <Menu.Item
              variant="danger"
              onClick={() => onAction({ kind: 'delete' })}
            >
              {t('deleteTitle')}
            </Menu.Item>
          </>
        )}
      </Menu.Popup>
    </Menu.Root>
  )
}

// 治理操作对话框状态机（含重置密码结果一次性展示，D3.3 方案 B）。
export function UserGovernanceDialogs({
  user,
  action,
  onClose,
}: {
  user: AdminUser
  action: PendingAction | null
  onClose: () => void
}) {
  const t = useTranslations('admin.users')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()
  const { copied, copy } = useCopyToClipboard(2000)

  const [deleteConfirm, setDeleteConfirm] = useState('')
  const [resetResult, setResetResult] = useState<string | null>(null)

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: adminKeys.users.all() })
    queryClient.invalidateQueries({ queryKey: adminKeys.stats() })
    queryClient.invalidateQueries({ queryKey: adminKeys.audit.all() })
    queryClient.invalidateQueries({ queryKey: adminKeys.lockedChannels() })
  }

  const close = () => {
    setDeleteConfirm('')
    setResetResult(null)
    onClose()
  }

  const fail = (err: Error) => {
    toastManager.add({ type: 'error', title: err.message })
  }

  const roleMutation = useMutation({
    mutationFn: (to: 'admin' | 'user') => adminApi.setUserRole(user.id, to),
    onSuccess: () => {
      invalidateAll()
      toastManager.add({ type: 'success', title: t('toasts.roleChanged') })
      close()
    },
    onError: fail,
  })

  const activeMutation = useMutation({
    mutationFn: (active: boolean) => adminApi.setUserActive(user.id, active),
    onSuccess: (_d, active) => {
      invalidateAll()
      toastManager.add({
        type: 'success',
        title: active
          ? t('toasts.enabled', { name: user.username })
          : t('toasts.disabled', { name: user.username }),
      })
      close()
    },
    onError: fail,
  })

  const logoutMutation = useMutation({
    mutationFn: () => adminApi.revokeAllSessions(user.id),
    onSuccess: () => {
      invalidateAll()
      queryClient.invalidateQueries({ queryKey: adminKeys.sessions(user.id) })
      toastManager.add({
        type: 'success',
        title: t('toasts.signOut', { name: user.username }),
      })
      close()
    },
    onError: fail,
  })

  const resetMutation = useMutation({
    mutationFn: () => adminApi.resetUserPassword(user.id),
    onSuccess: (data) => {
      invalidateAll()
      queryClient.invalidateQueries({ queryKey: adminKeys.sessions(user.id) })
      toastManager.add({ type: 'success', title: t('reset.success') })
      setResetResult(data.password)
    },
    onError: fail,
  })

  const deleteMutation = useMutation({
    mutationFn: () => adminApi.deleteUser(user.id),
    onSuccess: () => {
      invalidateAll()
      toastManager.add({ type: 'success', title: t('toasts.deleted') })
      close()
    },
    onError: fail,
  })

  const confirmLabel = (() => {
    switch (action?.kind) {
      case 'role':
        return action.to === 'admin'
          ? t('role.toAdmin.title')
          : t('role.toUser.title')
      case 'disable':
        return t('disable.title', { name: user.username })
      case 'enable':
        return t('enable.title', { name: user.username })
      case 'forceLogout':
        return t('logout.title', { name: user.username })
      case 'reset':
        return t('reset.title', { name: user.username })
      case 'delete':
        return t('delete.title', { name: user.username })
      default:
        return ''
    }
  })()

  const description = (() => {
    switch (action?.kind) {
      case 'role':
        return action.to === 'admin'
          ? t('role.toAdmin.desc', { name: user.username })
          : t('role.toUser.desc', { name: user.username })
      case 'disable':
        return t('disable.desc', { name: user.username })
      case 'enable':
        return t('enable.desc', { name: user.username })
      case 'forceLogout':
        return t('logout.desc', { name: user.username })
      case 'reset':
        return t('reset.desc', { name: user.username })
      case 'delete':
        return t('delete.desc', { name: user.username })
      default:
        return ''
    }
  })()

  const pending =
    roleMutation.isPending ||
    activeMutation.isPending ||
    logoutMutation.isPending ||
    resetMutation.isPending ||
    deleteMutation.isPending

  const confirm = () => {
    switch (action?.kind) {
      case 'role':
        roleMutation.mutate(action.to)
        break
      case 'disable':
        activeMutation.mutate(false)
        break
      case 'enable':
        activeMutation.mutate(true)
        break
      case 'forceLogout':
        logoutMutation.mutate()
        break
      case 'reset':
        resetMutation.mutate()
        break
      case 'delete':
        deleteMutation.mutate()
        break
    }
  }

  const isDelete = action?.kind === 'delete'
  const deleteDisabled = isDelete && deleteConfirm.trim() !== user.username

  return (
    <>
      <AlertDialog
        open={action !== null && action.kind !== 'reset'}
        onOpenChange={(open) => !open && close()}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirmLabel}</AlertDialogTitle>
            <AlertDialogDescription>{description}</AlertDialogDescription>
          </AlertDialogHeader>
          {isDelete && (
            <Input
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
              placeholder={t('delete.typePlaceholder')}
              autoFocus
              disabled={pending}
            />
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant={
                isDelete || action?.kind === 'disable' ? 'danger' : 'default'
              }
              disabled={pending || deleteDisabled}
              onClick={confirm}
            >
              {action?.kind === 'role'
                ? action.to === 'admin'
                  ? t('role.toAdmin.title')
                  : t('role.toUser.title')
                : isDelete
                  ? t('delete.permanent')
                  : tCommon('confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 重置密码确认 */}
      <AlertDialog
        open={action?.kind === 'reset'}
        onOpenChange={(open) => !open && close()}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirmLabel}</AlertDialogTitle>
            <AlertDialogDescription>{description}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction disabled={pending} onClick={confirm}>
              {t('reset.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* D3.3 方案 B：一次性密码结果 Dialog */}
      <Dialog
        open={resetResult !== null}
        onOpenChange={(open) => !open && close()}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('reset.result.title')}</DialogTitle>
            <DialogDescription>
              {t('reset.result.warning', { name: user.username })}
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-md border bg-muted px-3 py-2 font-mono text-sm">
              {resetResult}
            </code>
            <Button
              type="button"
              variant="outline"
              onClick={() => resetResult && copy(resetResult)}
            >
              {copied ? tCommon('copied') : t('reset.result.copy')}
            </Button>
          </div>
          <DialogFooter>
            <Button type="button" onClick={close}>
              {t('reset.result.close')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
