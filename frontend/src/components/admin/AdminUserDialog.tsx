'use client'

import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Field } from '@base-ui/react/field'
import { Form } from '@base-ui/react/form'
import { adminApi, adminKeys } from '@/lib/api'
import { adminCreateUserSchema } from '@/lib/schemas'
import { toastManager } from '@/stores/toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PasswordStrengthMeter } from '@/components/ui/password-strength-meter'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface AdminUserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface CreateUserValues {
  username: string
  email?: string
  password: string
}

// F.10 新建用户 Dialog：RHF + zod（用户名必填、邮箱可选但填了须 email、
// 密码 min 8/max 72）+ 强度指示器。
export function AdminUserDialog({ open, onOpenChange }: AdminUserDialogProps) {
  const t = useTranslations('admin.userDialog')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()

  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { isSubmitting },
  } = useForm<CreateUserValues>({
    resolver: zodResolver(adminCreateUserSchema),
    defaultValues: { username: '', email: '', password: '' },
    mode: 'onSubmit',
  })

  const { mutate } = useMutation({
    mutationFn: (values: CreateUserValues) =>
      adminApi.createUser({
        username: values.username,
        email: values.email ?? '',
        password: values.password,
      }),
    onSuccess: () => {
      reset()
      queryClient.invalidateQueries({ queryKey: adminKeys.users.all() })
      queryClient.invalidateQueries({ queryKey: adminKeys.stats() })
      toastManager.add({ type: 'success', title: t('created') })
      onOpenChange(false)
    },
    onError: (err: unknown) => {
      const apiErr = err as {
        fieldErrors?: { field?: string; message: string }[]
        message?: string
      }
      if (apiErr.fieldErrors?.length) {
        for (const fe of apiErr.fieldErrors) {
          if (fe.field === 'username')
            setError('username', { message: fe.message })
          else if (fe.field === 'email')
            setError('email', { message: fe.message })
          else if (fe.field === 'password')
            setError('password', { message: fe.message })
        }
        return
      }
      toastManager.add({ type: 'error', title: apiErr.message ?? '' })
    },
  })

  const fieldMsg = (
    message: string | undefined,
    field: 'username' | 'email' | 'password',
  ) => {
    if (!message) return ''
    switch (message) {
      case 'required':
        return field === 'username'
          ? t('usernameRequired')
          : field === 'password'
            ? t('passwordRequired')
            : ''
      case 'min_length':
        return t('passwordMin')
      case 'invalid_email':
        return t('emailInvalid')
      default:
        return message
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <Form
          onSubmit={handleSubmit((v) => mutate(v))}
          className="space-y-4"
          aria-label={t('title')}
        >
          <DialogHeader>
            <DialogTitle>{t('title')}</DialogTitle>
          </DialogHeader>

          <Controller
            name="username"
            control={control}
            render={({
              field: { ref, name, value, onChange, onBlur },
              fieldState: { invalid, isTouched, isDirty, error },
            }) => (
              <Field.Root
                name={name}
                invalid={invalid}
                touched={isTouched}
                dirty={isDirty}
              >
                <Field.Label>{t('username')}</Field.Label>
                <Field.Control
                  ref={ref}
                  value={value}
                  onValueChange={onChange}
                  onBlur={onBlur}
                  render={<Input autoComplete="off" autoFocus />}
                />
                <Field.Error match={!!error}>
                  {fieldMsg(error?.message, 'username')}
                </Field.Error>
              </Field.Root>
            )}
          />

          <Controller
            name="email"
            control={control}
            render={({
              field: { ref, name, value, onChange, onBlur },
              fieldState: { invalid, isTouched, isDirty, error },
            }) => (
              <Field.Root
                name={name}
                invalid={invalid}
                touched={isTouched}
                dirty={isDirty}
              >
                <Field.Label>{t('email')}</Field.Label>
                <Field.Control
                  ref={ref}
                  value={value}
                  onValueChange={onChange}
                  onBlur={onBlur}
                  render={<Input type="email" autoComplete="off" />}
                />
                <Field.Error match={!!error}>
                  {fieldMsg(error?.message, 'email')}
                </Field.Error>
              </Field.Root>
            )}
          />

          <Controller
            name="password"
            control={control}
            render={({
              field: { ref, name, value, onChange, onBlur },
              fieldState: { invalid, isTouched, isDirty, error },
            }) => (
              <Field.Root
                name={name}
                invalid={invalid}
                touched={isTouched}
                dirty={isDirty}
              >
                <Field.Label>{t('password')}</Field.Label>
                <Field.Control
                  ref={ref}
                  value={value}
                  onValueChange={onChange}
                  onBlur={onBlur}
                  type="password"
                  render={<Input autoComplete="new-password" />}
                />
                <Field.Description>
                  <PasswordStrengthMeter password={value} />
                </Field.Description>
                <Field.Error match={!!error}>
                  {fieldMsg(error?.message, 'password')}
                </Field.Error>
              </Field.Root>
            )}
          />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              {tCommon('cancel')}
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? tCommon('processing') : t('create')}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

export default AdminUserDialog
