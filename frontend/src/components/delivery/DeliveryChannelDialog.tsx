'use client'

import { useEffect, useRef, useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Field } from '@base-ui/react/field'
import { Form } from '@base-ui/react/form'
import { Trash2Icon } from 'lucide-react'
import { z } from 'zod'

import { adminKeys, deliveryApi, deliveryKeys } from '@/lib/api'
import { toastManager } from '@/stores/toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { feishuConfigurationSchema } from '@/utils/channel-form'
import { compileKeywordFilter } from '@/lib/keyword-filter'
import type { DeliveryChannel } from '@/types/delivery'

interface DeliveryChannelDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editingChannel: DeliveryChannel | null
}

// D5 渠道编辑 Dialog：base-ui Form/Field + RHF + zod。
// 校验时机（D5.2）：提交时全量；webhook_url onBlur；关键词 onChange 实时编译。
// 表单状态机（D5.3）：isSubmitting/confirmDelete 互斥禁用；测试独立状态机（D5.4）。
const channelFormSchema = z.object({
  name: z
    .string({ error: 'required' })
    .min(1, { error: 'required' })
    .max(64, { error: 'too_long' }),
  webhook_url: feishuConfigurationSchema.shape.webhook_url,
  card_link_url: feishuConfigurationSchema.shape.card_link_url,
  keywords: z.string(),
})

interface ChannelFormValues {
  name: string
  webhook_url: string
  card_link_url?: string
  keywords: string
}

type TestState = 'idle' | 'pending' | 'success' | 'failed'

export function DeliveryChannelDialog({
  open,
  onOpenChange,
  editingChannel,
}: DeliveryChannelDialogProps) {
  const t = useTranslations('delivery')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()

  const isEditing = editingChannel !== null

  const [confirmDelete, setConfirmDelete] = useState(false)
  const [testState, setTestState] = useState<TestState>('idle')
  const [testError, setTestError] = useState('')
  const testTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { isDirty },
  } = useForm<ChannelFormValues>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: {
      name: editingChannel?.name ?? '',
      webhook_url: editingChannel?.configuration?.webhook_url ?? '',
      card_link_url: editingChannel?.configuration?.card_link_url ?? '',
      keywords: editingChannel?.keywords ?? '',
    },
    mode: 'onSubmit',
  })

  // RHF 官方范式：Dialog 打开时用编辑对象重置表单（reset 属外部库状态同步）。
  useEffect(() => {
    reset(
      editingChannel
        ? {
            name: editingChannel.name,
            webhook_url: editingChannel.configuration?.webhook_url ?? '',
            card_link_url: editingChannel.configuration?.card_link_url ?? '',
            keywords: editingChannel.keywords,
          }
        : { name: '', webhook_url: '', card_link_url: '', keywords: '' },
    )
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setTestState('idle')
    setTestError('')
    setConfirmDelete(false)
  }, [open, editingChannel, reset])

  useEffect(() => {
    return () => {
      if (testTimerRef.current) clearTimeout(testTimerRef.current)
    }
  }, [])

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: deliveryKeys.channels() })
    queryClient.invalidateQueries({ queryKey: deliveryKeys.latest() })
    queryClient.invalidateQueries({ queryKey: deliveryKeys.history(1, 20) })
    // H.3 清单：渠道写操作联动 admin 统计与详情缓存。
    queryClient.invalidateQueries({ queryKey: adminKeys.stats() })
    if (editingChannel) {
      queryClient.invalidateQueries({
        queryKey: deliveryKeys.channelDetail(editingChannel.id),
      })
    }
  }

  const handleServerFieldErrors = (
    fieldErrors: { field?: string; message: string }[],
  ) => {
    for (const fe of fieldErrors) {
      if (fe.field === 'name') setError('name', { message: fe.message })
      else if (fe.field === 'webhook_url')
        setError('webhook_url', { message: fe.message })
      else if (fe.field === 'card_link_url')
        setError('card_link_url', { message: fe.message })
      else if (fe.field === 'keywords')
        setError('keywords', { message: fe.message })
    }
  }

  const saveMutation = useMutation({
    mutationFn: (values: ChannelFormValues) => {
      const payload = {
        kind: 'feishu',
        name: values.name,
        configuration: {
          webhook_url: values.webhook_url,
          card_link_url: values.card_link_url || '',
        },
        keywords: values.keywords,
      }
      return isEditing
        ? deliveryApi.update(editingChannel.id, payload)
        : deliveryApi.create(payload)
    },
    onSuccess: () => {
      invalidate()
      toastManager.add({
        type: 'success',
        title: isEditing ? t('dialog.updated') : t('dialog.created'),
      })
      onOpenChange(false)
    },
    onError: (err: unknown) => {
      const apiErr = err as {
        fieldErrors?: { field?: string; message: string }[]
        message?: string
      }
      if (apiErr.fieldErrors?.length) {
        handleServerFieldErrors(apiErr.fieldErrors)
        return
      }
      toastManager.add({
        type: 'error',
        title: apiErr.message ?? t('dialog.testFailed'),
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deliveryApi.delete(id),
    onSuccess: () => {
      invalidate()
      toastManager.add({ type: 'success', title: t('dialog.deleted') })
      onOpenChange(false)
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  // D5.4 测试投递状态机：idle → pending → success(3s 回 idle)/failed(重试)。
  const testMutation = useMutation({
    mutationFn: (id: number) => deliveryApi.test(id),
    onMutate: () => {
      setTestState('pending')
      setTestError('')
    },
    onSuccess: () => {
      setTestState('success')
      invalidate()
      toastManager.add({ type: 'success', title: t('dialog.testSuccess') })
      testTimerRef.current = setTimeout(() => setTestState('idle'), 3000)
    },
    onError: (err: Error) => {
      setTestState('failed')
      setTestError(err.message)
      toastManager.add({ type: 'error', title: t('dialog.testFailed') })
    },
  })

  const isSaving = saveMutation.isPending
  const inputsDisabled = isSaving || confirmDelete
  const testButtonDisabled = testState === 'pending'

  const fieldMsg = (
    message: string | undefined,
    field: 'name' | 'webhook' | 'card' = 'name',
  ) => {
    if (!message) return ''
    if (message === 'required')
      return field === 'webhook'
        ? t('dialog.webhookRequired')
        : field === 'card'
          ? t('dialog.webhookRequired')
          : t('dialog.nameRequired')
    if (message === 'invalid_url') return t('dialog.invalidUrl')
    if (message === 'too_long') return t('dialog.nameTooLong')
    return message
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-lg">
          <Form
            onSubmit={handleSubmit((v) => saveMutation.mutate(v))}
            className="space-y-4"
            aria-label={
              isEditing ? t('dialog.editTitle') : t('dialog.createTitle')
            }
          >
            <DialogHeader>
              <DialogTitle>
                {isEditing ? t('dialog.editTitle') : t('dialog.createTitle')}
              </DialogTitle>
            </DialogHeader>

            <div className="space-y-3">
              <Controller
                name="name"
                control={control}
                render={({
                  field: { ref, name, value, onChange, onBlur },
                  fieldState: { invalid, isTouched, isDirty: dirty, error },
                }) => (
                  <Field.Root
                    name={name}
                    invalid={invalid}
                    touched={isTouched}
                    dirty={dirty}
                  >
                    <Field.Label>{t('dialog.name')}</Field.Label>
                    <Field.Control
                      ref={ref}
                      value={value}
                      onValueChange={onChange}
                      onBlur={onBlur}
                      render={
                        <Input autoComplete="off" autoFocus maxLength={64} />
                      }
                      placeholder={t('dialog.namePlaceholder')}
                      disabled={inputsDisabled}
                    />
                    <Field.Error match={!!error}>
                      {fieldMsg(error?.message)}
                    </Field.Error>
                  </Field.Root>
                )}
              />

              <Controller
                name="webhook_url"
                control={control}
                render={({
                  field: { ref, name, value, onChange, onBlur },
                  fieldState: { invalid, isTouched, isDirty: dirty, error },
                }) => (
                  <Field.Root
                    name={name}
                    invalid={invalid}
                    touched={isTouched}
                    dirty={dirty}
                    validationMode="onBlur"
                  >
                    <Field.Label>{t('dialog.webhookURL')}</Field.Label>
                    <Field.Control
                      ref={ref}
                      value={value}
                      onValueChange={onChange}
                      onBlur={onBlur}
                      render={
                        <Input
                          type="url"
                          inputMode="url"
                          autoComplete="off"
                          spellCheck={false}
                        />
                      }
                      placeholder={t('dialog.webhookPlaceholder')}
                      disabled={inputsDisabled}
                    />
                    <Field.Error match={!!error}>
                      {fieldMsg(error?.message, 'webhook')}
                    </Field.Error>
                  </Field.Root>
                )}
              />

              <Controller
                name="card_link_url"
                control={control}
                render={({
                  field: { ref, name, value, onChange, onBlur },
                  fieldState: { invalid, isTouched, isDirty: dirty, error },
                }) => (
                  <Field.Root
                    name={name}
                    invalid={invalid}
                    touched={isTouched}
                    dirty={dirty}
                  >
                    <Field.Label>{t('dialog.cardLinkURL')}</Field.Label>
                    <Field.Control
                      ref={ref}
                      value={value}
                      onValueChange={onChange}
                      onBlur={onBlur}
                      render={<Input type="url" autoComplete="off" />}
                      placeholder={t('dialog.cardLinkURLPlaceholder')}
                      disabled={inputsDisabled}
                    />
                    <Field.Error match={!!error}>
                      {fieldMsg(error?.message, 'card')}
                    </Field.Error>
                  </Field.Root>
                )}
              />

              <KeywordField
                control={control}
                disabled={inputsDisabled}
                fieldMsg={fieldMsg}
              />
            </div>

            <DialogFooter className="sm:justify-between">
              {isEditing && (
                <div className="flex items-center gap-2">
                  {/* D5.4 测试按钮状态机 */}
                  <Button
                    type="button"
                    variant={testState === 'failed' ? 'danger' : 'outline'}
                    disabled={testButtonDisabled || inputsDisabled}
                    onClick={() =>
                      editingChannel && testMutation.mutate(editingChannel.id)
                    }
                    title={testError}
                  >
                    {testState === 'pending'
                      ? t('dialog.testing')
                      : testState === 'success'
                        ? t('dialog.testSent')
                        : testState === 'failed'
                          ? t('dialog.testRetry')
                          : t('dialog.test')}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    className="text-danger hover:text-danger"
                    onClick={() => setConfirmDelete(true)}
                    disabled={isSaving}
                  >
                    <Trash2Icon className="mr-1 size-4" />
                    {t('dialog.delete')}
                  </Button>
                </div>
              )}

              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => onOpenChange(false)}
                  disabled={inputsDisabled}
                >
                  {t('dialog.cancel')}
                </Button>
                <Button
                  type="submit"
                  disabled={inputsDisabled || (isEditing && !isDirty)}
                >
                  {isSaving
                    ? t('dialog.saving')
                    : isEditing
                      ? t('dialog.save')
                      : t('dialog.create')}
                </Button>
              </div>
            </DialogFooter>
          </Form>
        </DialogContent>
      </Dialog>

      {/* D5.5 删除统一 AlertDialog（防误删，后果如实告知） */}
      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('dialog.deleteConfirm.title')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t('dialog.deleteConfirm.desc', {
                name: editingChannel?.name ?? '',
              })}
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
                editingChannel && deleteMutation.mutate(editingChannel.id)
              }
            >
              {deleteMutation.isPending
                ? t('dialog.deleting')
                : t('dialog.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

// D5.6 关键词过滤：onChange 实时编译（保留语法校验，移除语义解释），
// [?] hover（桌面）+ tap（移动）展开完整语法。
function KeywordField({
  control,
  disabled,
  fieldMsg,
}: {
  control: ReturnType<typeof useForm<ChannelFormValues>>['control']
  disabled: boolean
  fieldMsg: (m: string | undefined) => string
}) {
  const t = useTranslations('delivery')
  const [syntaxHelpOpen, setSyntaxHelpOpen] = useState(false)

  return (
    <Controller
      name="keywords"
      control={control}
      render={({
        field: { ref, name, value, onChange, onBlur },
        fieldState: { invalid, isTouched, isDirty: dirty, error },
      }) => {
        const { error: compileError } = compileKeywordFilter(value)
        return (
          <Field.Root
            name={name}
            invalid={invalid || !!compileError}
            touched={isTouched}
            dirty={dirty}
            validationMode="onChange"
          >
            <Field.Label className="gap-1">
              {t('dialog.keywords')}
              <span className="relative inline-flex">
                <button
                  type="button"
                  tabIndex={-1}
                  aria-label={t('dialog.keywordsSyntax')}
                  aria-expanded={syntaxHelpOpen}
                  onClick={() => setSyntaxHelpOpen((v) => !v)}
                  className="flex size-4 items-center justify-center rounded-full bg-muted text-xs text-muted-foreground"
                >
                  ?
                </button>
                {/* D5.6：hover（桌面）与 tap（移动）均可展开语法帮助 */}
                <span
                  className={`pointer-events-none absolute top-full left-0 z-10 mt-1 w-56 rounded-md border bg-popover p-2 text-xs text-popover-foreground shadow-lg ${syntaxHelpOpen ? 'block' : 'hidden group-hover:block'}`}
                >
                  {t('dialog.keywordsSyntax')}
                </span>
              </span>
            </Field.Label>
            <Field.Control
              ref={ref}
              value={value}
              onValueChange={onChange}
              onBlur={onBlur}
              render={<Input autoComplete="off" spellCheck={false} />}
              placeholder={t('dialog.keywordsPlaceholder')}
              disabled={disabled}
            />
            <Field.Description>{t('dialog.keywordsHelp')}</Field.Description>
            <Field.Error match={!!error || !!compileError}>
              {compileError
                ? t('dialog.keywordsInvalid', { error: compileError })
                : fieldMsg(error?.message)}
            </Field.Error>
          </Field.Root>
        )
      }}
    />
  )
}

export default DeliveryChannelDialog
