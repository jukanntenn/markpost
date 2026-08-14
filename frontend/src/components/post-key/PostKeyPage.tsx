'use client'

import { useSyncExternalStore, useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { authApi, postKeyKeys } from '@/lib/api'
import { usePostKey } from '@/hooks/usePostKey'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { toastManager } from '@/stores/toast'
import { buildFullPostUrl } from '@/utils/url'
import { PageHeading } from '@/components/ui/page-heading'
import { ListState } from '@/components/ui/list-state'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
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
  CheckIcon,
  ChevronDownIcon,
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  KeyRoundIcon,
  LinkIcon,
  TerminalIcon,
} from 'lucide-react'
import { formatToLocalTime } from '@/utils/time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import CreateTestPostModal from '@/components/CreateTestPostModal'

// A2.9/F.1 Post Key 页：凭证展示 + 轮换（C2.5）+ 测试发帖 + 发送文档聚合。
export function PostKeyPage() {
  const t = useTranslations('postKey')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()
  const queryClient = useQueryClient()

  const query = usePostKey()
  const { copied: copiedField, copy: copyField } = useCopyToClipboard(1500)
  const { copied: copiedCurl, copy: copyCurl } = useCopyToClipboard(1500)

  const [showKey, setShowKey] = useState(false)
  const [confirmRotate, setConfirmRotate] = useState(false)
  const [showTestModal, setShowTestModal] = useState(false)
  // window.location.origin is client-only; useSyncExternalStore returns '' on
  // the server snapshot to avoid hydration mismatch, and the live value on the
  // client — without a setState-in-effect.
  const origin = useSyncExternalStore(
    () => () => {},
    () => window.location.origin,
    () => '',
  )

  const postKey = query.data?.post_key ?? ''
  const createdAt = query.data?.created_at ?? ''
  const postUrl = buildFullPostUrl(postKey)
  const maskedKey = '•'.repeat(Math.min(postKey.length, 16))

  // C2.5 轮换：rotating ref 防双击（token_version 无涉，但防重复轮换）。
  const rotateMutation = useMutation({
    mutationFn: () => authApi.rotatePostKey(),
    onSuccess: () => {
      setConfirmRotate(false)
      setShowKey(true)
      toastManager.add({ type: 'success', title: t('rotateSuccess') })
      queryClient.invalidateQueries({ queryKey: postKeyKeys.current() })
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  const copyWithFeedback = (text: string) => {
    if (!text) return
    copyField(text).then((ok) => {
      if (!ok) toastManager.add({ type: 'error', title: t('copyFailed') })
    })
  }
  const onCopyUrl = () => copyWithFeedback(postUrl)
  const onCopyKey = () => copyWithFeedback(postKey)

  const onCopyCurl = () => {
    const curl = `curl -X POST ${postUrl} \\\n  -H 'Content-Type: application/json' \\\n  -d '{"title":"Hello","body":"**World**"}'`
    copyCurl(curl).then((ok) => {
      if (!ok) toastManager.add({ type: 'error', title: t('copyFailed') })
    })
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeading>{t('title')}</PageHeading>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRoundIcon className="size-4 text-primary" />
            {t('title')}
          </CardTitle>
          <CardDescription>{t('explanation')}</CardDescription>
        </CardHeader>
        <CardContent>
          <ListState
            isLoading={query.isLoading}
            error={query.error}
            loadingSkeleton={
              <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-10 w-64" />
              </div>
            }
            onRetry={() => query.refetch()}
          >
            <div className="flex items-center gap-2">
              <code className="flex min-w-0 flex-1 items-center rounded-md border bg-muted px-3 py-2 font-mono text-sm">
                <span
                  className="truncate"
                  title={showKey ? postUrl : undefined}
                >
                  {origin}/{showKey ? postKey : maskedKey}
                </span>
              </code>
              <Button
                type="button"
                variant="outline"
                size="icon"
                aria-label={showKey ? t('hideKey') : t('showKey')}
                onClick={() => setShowKey((v) => !v)}
              >
                {showKey ? (
                  <EyeOffIcon className="size-4" />
                ) : (
                  <EyeIcon className="size-4" />
                )}
              </Button>
              <div className="flex">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="rounded-r-none"
                  aria-label={t('copyPostUrl')}
                  onClick={onCopyUrl}
                  disabled={!postUrl}
                >
                  {copiedField ? (
                    <CheckIcon className="size-4 text-success" />
                  ) : (
                    <CopyIcon className="size-4" />
                  )}
                </Button>
                <Menu>
                  <Menu.Trigger
                    render={
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        className="-ml-px rounded-l-none"
                        aria-label={t('copyOptions')}
                      />
                    }
                  >
                    <ChevronDownIcon className="size-4" />
                  </Menu.Trigger>
                  <Menu.Popup>
                    <Menu.Item onClick={onCopyUrl}>
                      <LinkIcon className="size-4" />
                      {t('copyPostUrl')}
                    </Menu.Item>
                    <Menu.Item onClick={onCopyKey}>
                      <KeyRoundIcon className="size-4" />
                      {t('copyKey')}
                    </Menu.Item>
                  </Menu.Popup>
                </Menu>
              </div>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {t('createdAt')}:{' '}
              {formatToLocalTime(createdAt, { includeSeconds: false, locale })}
            </p>

            <Separator className="my-4" />

            <div className="flex flex-wrap items-center gap-2">
              <Button type="button" onClick={() => setShowTestModal(true)}>
                {t('testPost')}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => setConfirmRotate(true)}
              >
                {t('rotate')}
              </Button>
              <span className="text-xs text-muted-foreground">
                {t('rotateConfirm.desc').split('。')[0]}。
              </span>
            </div>
          </ListState>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TerminalIcon className="size-4 text-primary" />
            {t('howToSend.title')}
          </CardTitle>
          <CardDescription>{t('howToSend.desc')}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <pre className="overflow-x-auto rounded-md border bg-muted p-4 font-mono text-xs leading-relaxed">
              {`POST ${buildFullPostUrl(postKey)}\nContent-Type: application/json\n\n{\n  "title": "...",\n  "body": "..."\n}`}
            </pre>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={onCopyCurl}
              >
                {copiedCurl ? (
                  <CheckIcon className="size-4 text-success" />
                ) : (
                  <CopyIcon className="size-4" />
                )}
                {t('howToSend.copyCurl')}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <AlertDialog open={confirmRotate} onOpenChange={setConfirmRotate}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('rotateConfirm.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('rotateConfirm.desc')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={rotateMutation.isPending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={rotateMutation.isPending}
              onClick={() => rotateMutation.mutate()}
            >
              {t('rotate')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {postKey && (
        <CreateTestPostModal
          show={showTestModal}
          postKey={postKey}
          onHide={() => setShowTestModal(false)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['posts'] })
            queryClient.invalidateQueries({ queryKey: ['delivery'] })
          }}
        />
      )}
    </div>
  )
}

export default PostKeyPage
