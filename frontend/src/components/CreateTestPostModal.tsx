'use client'

import { useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Field } from '@base-ui/react/field'
import { Form } from '@base-ui/react/form'
import { toastManager } from '@/stores/toast'
import { request } from '@/lib/api'
import { createTestPostSchema } from '@/lib/schemas'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import type {
  CreateTestPostRequest,
  CreateTestPostResponse,
} from '@/types/posts'

interface CreateTestPostModalProps {
  show: boolean
  postKey: string
  onHide: () => void
  onSuccess: () => void
}

interface TestPostValues {
  title: string
  body: string
}

async function createTestPost(
  postKey: string,
  data: CreateTestPostRequest,
): Promise<CreateTestPostResponse> {
  return request<CreateTestPostResponse>(`/${postKey}`, {
    method: 'POST',
    json: data,
    skipAuthRefresh: true,
  })
}

// F.11 测试发帖 Dialog：base-ui Form + RHF + zod（title/body 必填，
// title ≤150 字符）。成功 toast + invalidate 活动流（H.3 清单）。
function CreateTestPostModal({
  show,
  postKey,
  onHide,
  onSuccess,
}: CreateTestPostModalProps) {
  const t = useTranslations('createTestPost')

  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = useForm<TestPostValues>({
    resolver: zodResolver(createTestPostSchema),
    defaultValues: { title: '', body: '' },
    mode: 'onSubmit',
  })

  useEffect(() => {
    if (show) reset()
  }, [show, reset])

  const { mutate } = useMutation({
    mutationFn: (data: TestPostValues) =>
      createTestPost(postKey, {
        title: data.title.trim() || 'Untitled',
        body: data.body,
      }),
    onSuccess: () => {
      toastManager.add({ type: 'success', title: t('success') })
      reset()
      onSuccess()
      onHide()
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  const fieldMessage = (
    message: string | undefined,
    field: 'title' | 'body',
  ) => {
    if (!message) return ''
    if (message === 'required')
      return field === 'title' ? t('titleRequired') : t('bodyRequired')
    if (message === 'title_too_long') return t('titleTooLong')
    return t(message)
  }

  return (
    <Dialog open={show} onOpenChange={(open) => !open && onHide()}>
      <DialogContent className="sm:max-w-2xl">
        <Form
          onSubmit={handleSubmit((v) => mutate(v))}
          className="space-y-4"
          aria-label={t('title')}
        >
          <DialogHeader>
            <DialogTitle>{t('title')}</DialogTitle>
          </DialogHeader>

          <Controller
            name="title"
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
                <Field.Label>{t('titleLabel')}</Field.Label>
                <Field.Control
                  ref={ref}
                  value={value}
                  onValueChange={onChange}
                  onBlur={onBlur}
                  render={<Input autoComplete="off" autoFocus />}
                  placeholder={t('titlePlaceholder')}
                />
                <Field.Error match={!!error}>
                  {fieldMessage(error?.message, 'title')}
                </Field.Error>
              </Field.Root>
            )}
          />

          <Controller
            name="body"
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
                <Field.Label>{t('bodyLabel')}</Field.Label>
                <Field.Control
                  ref={ref}
                  value={value}
                  onValueChange={onChange}
                  onBlur={onBlur}
                  render={<Textarea rows={8} />}
                  placeholder={t('bodyPlaceholder')}
                />
                <Field.Error match={!!error}>
                  {fieldMessage(error?.message, 'body')}
                </Field.Error>
              </Field.Root>
            )}
          />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={onHide}
              disabled={isSubmitting}
            >
              {t('cancel')}
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? t('creating') : t('create')}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

export default CreateTestPostModal
