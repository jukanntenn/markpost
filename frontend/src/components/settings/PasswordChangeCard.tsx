'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm, type Control } from 'react-hook-form'
import { Field } from '@base-ui/react/field'
import { Form } from '@base-ui/react/form'
import { EyeIcon, EyeOffIcon } from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { FormAlert } from '@/components/ui/form-alert'
import { PasswordStrengthMeter } from '@/components/ui/password-strength-meter'
import { Separator } from '@/components/ui/separator'
import { authApi, ApiError, ApiErrorCodes } from '@/lib/api'
import { passwordChangeSchema } from '@/lib/schemas'
import { useAuthStore } from '@/stores/auth'
import { toastManager } from '@/stores/toast'

interface PasswordFormValues {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

// B1.11 改密流程：base-ui Form/Field + RHF + zod + 强度指示器 + 吊销联动。
// 成功后 setTokens（不强制重登，无缝继续）；反馈统一 toast（收敛 Alert+setTimeout）。
// F.2 裁决：作为"安全"卡片内嵌区块（embedded）时去掉自身 Card 壳。
export function PasswordChangeCard({
  embedded = false,
}: {
  embedded?: boolean
}) {
  const t = useTranslations('settings.password')
  const tCommon = useTranslations('common')
  const tErr = useTranslations('passwordChange.error')
  const tNetwork = useTranslations('network')
  const setTokens = useAuthStore((state) => state.setTokens)

  const [formAlert, setFormAlert] = useState('')
  const [visible, setVisible] = useState<Record<string, boolean>>({})

  const {
    control,
    handleSubmit,
    setError,
    reset,
    formState: { isSubmitting },
  } = useForm<PasswordFormValues>({
    resolver: zodResolver(passwordChangeSchema),
    defaultValues: {
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
    },
    mode: 'onSubmit',
  })

  const toggleVisible = (field: string) =>
    setVisible((v) => ({ ...v, [field]: !v[field] }))

  const { mutate } = useMutation({
    mutationFn: (values: PasswordFormValues) =>
      authApi.changePassword(values.currentPassword, values.newPassword),
    onSuccess: (data) => {
      // C2.2 吊销联动：新 token 对替换本地（旧 token 已被 token_version 作废）。
      setTokens(data.token, data.refresh_token)
      reset()
      setFormAlert('')
      toastManager.add({ type: 'success', title: t('success') })
    },
    onError: (err: unknown) => {
      const apiErr = err instanceof ApiError ? err : null

      // 字段级：服务端 fieldErrors 回灌。
      if (apiErr?.fieldErrors?.length) {
        for (const fe of apiErr.fieldErrors) {
          if (fe.field === 'current_password') {
            setError('currentPassword', { message: fe.message })
          } else if (fe.field === 'new_password') {
            setError('newPassword', { message: fe.message })
          } else if (fe.field === 'confirm_password') {
            setError('confirmPassword', { message: fe.message })
          }
        }
      }

      // 明确码映射（B1.8 场景 C）。
      switch (apiErr?.code) {
        case ApiErrorCodes.InvalidPassword:
          setError('currentPassword', { message: tErr('invalid_password') })
          break
        case ApiErrorCodes.RateLimited:
          toastManager.add({
            type: 'warning',
            title: apiErr.retryAfter
              ? `${tErr('rate_limited')} ${apiErr.retryAfter}s`
              : tErr('rate_limited'),
          })
          break
        case ApiErrorCodes.NetworkError:
        case ApiErrorCodes.Timeout:
          setFormAlert(
            apiErr?.code === ApiErrorCodes.Timeout
              ? tNetwork('timeout')
              : tNetwork('offline'),
          )
          break
        case ApiErrorCodes.Internal:
        case undefined:
          toastManager.add({ type: 'error', title: tErr('internal') })
          break
        default:
          if (!apiErr?.fieldErrors?.length) {
            setFormAlert(err instanceof Error ? err.message : String(err))
          }
      }
    },
  })

  const form = (
    <>
      <FormAlert message={formAlert} />
      <Form
        onSubmit={handleSubmit((v) => mutate(v))}
        className="mt-4 space-y-4"
      >
        <PasswordChangeFields
          control={control}
          visible={visible}
          onToggleVisible={toggleVisible}
        />
        <div className="flex justify-end gap-2 pt-2">
          {/* B1.11 [Cancel]：清空表单（配合 Change 按钮） */}
          <Button
            type="button"
            variant="ghost"
            disabled={isSubmitting}
            onClick={() => {
              reset()
              setFormAlert('')
            }}
          >
            {tCommon('cancel')}
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? t('changing') : t('submit')}
          </Button>
        </div>
      </Form>
    </>
  )

  if (embedded) {
    return <section aria-label={t('submit')}>{form}</section>
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('submit')}</CardTitle>
        <CardDescription>{t('currentHelp')}</CardDescription>
      </CardHeader>
      <CardContent>{form}</CardContent>
    </Card>
  )
}

function PasswordChangeFields({
  control,
  visible,
  onToggleVisible,
}: {
  control: Control<PasswordFormValues>
  visible: Record<string, boolean>
  onToggleVisible: (field: string) => void
}) {
  const t = useTranslations('settings.password')

  return (
    <>
      <Controller
        name="currentPassword"
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
            <Field.Label>{t('current')}</Field.Label>
            <div className="relative">
              <Field.Control
                ref={ref}
                value={value}
                onValueChange={onChange}
                onBlur={onBlur}
                type={visible.current ? 'text' : 'password'}
                render={<Input autoComplete="current-password" />}
                placeholder={t('currentPlaceholder')}
                className="pr-12"
              />
              <PasswordToggle
                shown={!!visible.current}
                label={visible.current ? t('hide') : t('show')}
                onClick={() => onToggleVisible('current')}
              />
            </div>
            <Field.Error match={!!error}>{error?.message ?? ''}</Field.Error>
          </Field.Root>
        )}
      />

      <Separator />

      <Controller
        name="newPassword"
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
            <Field.Label>{t('new')}</Field.Label>
            <div className="relative">
              <Field.Control
                ref={ref}
                value={value}
                onValueChange={onChange}
                onBlur={onBlur}
                type={visible.new ? 'text' : 'password'}
                render={<Input autoComplete="new-password" />}
                placeholder={t('newPlaceholder')}
                className="pr-12"
              />
              <PasswordToggle
                shown={!!visible.new}
                label={visible.current ? t('hide') : t('show')}
                onClick={() => onToggleVisible('new')}
              />
            </div>
            {/* B1.11 强度指示器：信息提示非错误，不阻断提交 */}
            <Field.Description>
              <PasswordStrengthMeter password={value} />
            </Field.Description>
            <Field.Error match={!!error}>
              {error
                ? error.message === 'min_length'
                  ? t('minLength')
                  : error.message === 'max_length'
                    ? t('maxLength')
                    : error.message
                : ''}
            </Field.Error>
          </Field.Root>
        )}
      />

      <Controller
        name="confirmPassword"
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
            <Field.Label>{t('confirm')}</Field.Label>
            <div className="relative">
              <Field.Control
                ref={ref}
                value={value}
                onValueChange={onChange}
                onBlur={onBlur}
                type={visible.confirm ? 'text' : 'password'}
                render={<Input autoComplete="new-password" />}
                placeholder={t('confirmPlaceholder')}
                className="pr-12"
              />
              <PasswordToggle
                shown={!!visible.confirm}
                label={visible.current ? t('hide') : t('show')}
                onClick={() => onToggleVisible('confirm')}
              />
            </div>
            <Field.Error match={!!error}>
              {error?.message === 'not_match'
                ? t('notMatch')
                : (error?.message ?? '')}
            </Field.Error>
          </Field.Root>
        )}
      />
    </>
  )
}

function PasswordToggle({
  shown,
  label,
  onClick,
}: {
  shown: boolean
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="absolute top-1/2 right-3 flex size-11 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
    >
      {shown ? (
        <EyeOffIcon className="size-4" />
      ) : (
        <EyeIcon className="size-4" />
      )}
    </button>
  )
}
