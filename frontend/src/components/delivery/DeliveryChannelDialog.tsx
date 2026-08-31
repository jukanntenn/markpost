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
import {
  compileKeywordFilter,
  describeFilter,
  type FilterError,
  type Phrasebook,
} from '@/lib/keyword-filter'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'
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

// D5.6 关键词过滤：onChange 实时编译 + 自然语言预览（cron-guru 式，
// 兑现 specs/backend/keyword-filter.md 承诺的全角标点防坑 UX），
// [?] Popover 承载语法速查与可点击试写的示例。
const KEYWORD_EXAMPLES = [
  'alert',
  'error, warning',
  'prod & !debug',
  '(error, warning) & prod',
  '"a,b"',
]

const TOKEN_CHARS: Record<string, string> = {
  comma: ',',
  pipe: '|',
  amp: '&',
  not: '!',
  lparen: '(',
  rparen: ')',
}

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

  const phr: Phrasebook = {
    quote: (kw) => t('dialog.keywordsPhQuote', { kw }),
    contains: (kw) => t('dialog.keywordsPhContains', { kw }),
    notContains: (kw) => t('dialog.keywordsPhNotContains', { kw }),
    notGroup: (inner) => t('dialog.keywordsPhNotGroup', { inner }),
    and: (a, b) => t('dialog.keywordsPhAnd', { a, b }),
    or: (a, b) => t('dialog.keywordsPhOr', { a, b }),
    group: (inner) => t('dialog.keywordsPhGroup', { inner }),
  }

  // 结构化解析错误 → 本地化文案；eof 单独处理（没有字符可指）。
  const errorMessage = (err: FilterError): string => {
    const tok =
      err.token && err.token !== 'eof' ? (TOKEN_CHARS[err.token] ?? '') : ''
    switch (err.code) {
      case 'unterminated_quote':
        return t('dialog.keywordsErrQuote', { pos: err.pos ?? 0 })
      case 'empty_keyword':
        return t('dialog.keywordsErrEmptyKw')
      case 'missing_rparen':
        return err.token === 'eof'
          ? t('dialog.keywordsErrRparenEnd')
          : t('dialog.keywordsErrRparen', { pos: err.pos ?? 0, token: tok })
      default:
        return err.token === 'eof'
          ? t('dialog.keywordsErrIncomplete')
          : t('dialog.keywordsErrUnexpected', { pos: err.pos ?? 0, token: tok })
    }
  }

  return (
    <Controller
      name="keywords"
      control={control}
      render={({
        field: { ref, name, value, onChange, onBlur },
        fieldState: { invalid, isTouched, isDirty: dirty, error },
      }) => {
        const { node, error: compileError } = compileKeywordFilter(value)
        const clause = describeFilter(node, phr)
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
              <KeywordSyntaxHelp disabled={disabled} onPickExample={onChange} />
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
            {!compileError && (
              <Field.Description className="text-muted-foreground">
                {clause === null
                  ? t('dialog.keywordsPreviewAlways')
                  : t('dialog.keywordsPreviewSentence', { expr: clause })}
              </Field.Description>
            )}
            <Field.Error match={!!error || !!compileError}>
              {compileError
                ? errorMessage(compileError)
                : fieldMsg(error?.message)}
            </Field.Error>
          </Field.Root>
        )
      }}
    />
  )
}

function KeywordSyntaxHelp({
  disabled,
  onPickExample,
}: {
  disabled: boolean
  onPickExample: (expr: string) => void
}) {
  const t = useTranslations('delivery')
  // 受控开合：点示例即关闭，让用户立刻看到填入的表达式与预览。
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={disabled}
        aria-label={t('dialog.keywordsSyntax')}
        className="flex size-4 items-center justify-center rounded-full bg-muted text-xs font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        ?
      </PopoverTrigger>
      <PopoverContent>
        <p className="text-[13px] font-semibold">
          {t('dialog.keywordsSyntaxTitle')}
        </p>
        <ul className="mt-1.5 space-y-1">
          <li>{t('dialog.keywordsHelpOr')}</li>
          <li>{t('dialog.keywordsHelpAnd')}</li>
          <li>{t('dialog.keywordsHelpNot')}</li>
          <li>{t('dialog.keywordsHelpGroup')}</li>
          <li>{t('dialog.keywordsHelpQuote')}</li>
        </ul>
        <ul className="mt-2 space-y-1 text-muted-foreground">
          <li>{t('dialog.keywordsNoteSpaces')}</li>
          <li>{t('dialog.keywordsNoteFullwidth')}</li>
          <li>{t('dialog.keywordsNoteEmpty')}</li>
        </ul>
        <p className="mt-2.5 font-medium">{t('dialog.keywordsTry')}</p>
        <div className="mt-1.5 flex flex-wrap gap-1.5">
          {KEYWORD_EXAMPLES.map((ex) => (
            <button
              key={ex}
              type="button"
              onClick={() => {
                onPickExample(ex)
                setOpen(false)
              }}
              className="rounded border bg-background px-1.5 py-0.5 font-mono hover:bg-accent hover:text-accent-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              {ex}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}

export default DeliveryChannelDialog
