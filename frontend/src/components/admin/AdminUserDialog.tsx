'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'

import { adminApi, adminKeys, invalidateKey } from '@/lib/api'
import { mutationOptions } from '@/lib/mutation-helpers'
import { toast } from '@/stores/toast'
import { Button } from '@/components/ui/button'
import { LoadingButton } from '@/components/ui/loading-button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface AdminUserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface UserFormState {
  email: string
  username: string
  password: string
}

const EMPTY_FORM: UserFormState = {
  email: '',
  username: '',
  password: '',
}

export function AdminUserDialog({ open, onOpenChange }: AdminUserDialogProps) {
  const t = useTranslations('admin')
  const queryClient = useQueryClient()
  const [form, setForm] = useState<UserFormState>(EMPTY_FORM)
  const [errors, setErrors] = useState<Partial<UserFormState>>({})

  function validate(): boolean {
    const newErrors: Partial<UserFormState> = {}
    if (!form.email) newErrors.email = t('users.emailRequired')
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
      newErrors.email = t('users.emailInvalid')
    if (!form.username) newErrors.username = t('users.usernameRequired')
    if (!form.password) newErrors.password = t('users.passwordRequired')
    else if (form.password.length < 6)
      newErrors.password = t('users.passwordMinLength')
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const createMutation = useMutation(
    mutationOptions({
      mutationFn: adminApi.createUser,
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.users.all())
        invalidateKey(queryClient, adminKeys.stats())
        toast.success(t('users.userCreated'))
        onOpenChange(false)
        setForm(EMPTY_FORM)
        setErrors({})
      },
    })
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    createMutation.mutate(form)
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) {
      setForm(EMPTY_FORM)
      setErrors({})
    }
    onOpenChange(isOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('users.addUser')}</DialogTitle>
          <DialogDescription>{t('users.addUserDescription')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">{t('users.email')}</Label>
            <Input
              id="email"
              type="email"
              placeholder={t('users.emailPlaceholder')}
              value={form.email}
              onChange={(e) =>
                setForm((prev) => ({ ...prev, email: e.target.value }))
              }
            />
            {errors.email && (
              <p className="text-sm text-destructive">{errors.email}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="username">{t('username')}</Label>
            <Input
              id="username"
              placeholder={t('users.usernamePlaceholder')}
              value={form.username}
              onChange={(e) =>
                setForm((prev) => ({ ...prev, username: e.target.value }))
              }
            />
            {errors.username && (
              <p className="text-sm text-destructive">{errors.username}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">{t('users.password')}</Label>
            <Input
              id="password"
              type="password"
              placeholder={t('users.passwordPlaceholder')}
              value={form.password}
              onChange={(e) =>
                setForm((prev) => ({ ...prev, password: e.target.value }))
              }
            />
            {errors.password && (
              <p className="text-sm text-destructive">{errors.password}</p>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                onOpenChange(false)
                setForm(EMPTY_FORM)
                setErrors({})
              }}
            >
              {t('users.cancel')}
            </Button>
            <LoadingButton
              type="submit"
              loading={createMutation.isPending}
              loadingText={t('users.creating')}
              disabled={createMutation.isPending}
            >
              {t('users.create')}
            </LoadingButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
