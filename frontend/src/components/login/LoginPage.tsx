'use client'

import { useEffect, useRef, useState } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Field } from '@base-ui/react/field'
import { Form } from '@base-ui/react/form'
import { GithubIcon, EyeIcon, EyeOffIcon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth'
import { authApi, ApiError, ApiErrorCodes } from '@/lib/api'
import { useGitHubOAuth } from '@/hooks/useGitHubOAuth'
import { loginSchema } from '@/lib/schemas'
import { safeNext } from '@/utils/safe-next'
import { toastManager } from '@/stores/toast'
import { FormAlert } from '@/components/ui/form-alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

interface LoginFormValues {
  username: string
  password: string
}

// B1.7 登录页：base-ui Form/Field + RHF + zod。实时只校验 required
// （不做长度校验，B1.7）。错误分三层（B1.6）：
//   表单级 FormAlert（invalid_credentials/user_disabled/account_locked/
//   internal/网络）；字段级 Field.Error（required / 服务端 fieldErrors）。
export default function LoginPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const t = useTranslations('login')
  const tCommon = useTranslations('common')
  const setAuth = useAuthStore((state) => state.setAuth)
  const { startOAuth, loading: loadingGitHub } = useGitHubOAuth()

  const [showPassword, setShowPassword] = useState(false)
  const [formAlert, setFormAlert] = useState('')
  const processingRef = useRef(false)

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' },
    mode: 'onSubmit',
  })

  // B1 场景 D：会话失效跳转带 ?reason=session_expired。
  const sessionExpired = searchParams.get('reason') === 'session_expired'
  // B1 场景 B2：OAuth 回调错误透传（?error=<code>）。
  const oauthErrorCode = searchParams.get('error')
  const next = searchParams.get('next')

  useEffect(() => {
    document.title = tCommon('pageTitle.login')
  }, [tCommon])

  const tCallback = useTranslations('loginCallback')

  // OAuth 回调错误 → 表单级提示（B1.8 场景 B 文案表）。
  const oauthAlert = (() => {
    if (!oauthErrorCode) return ''
    if (oauthErrorCode === 'user_disabled') return t('formAlert.user_disabled')
    const message = tCallback(`errors.${oauthErrorCode}`)
    return message.startsWith('errors.') ? '' : message
  })()

  const translateFieldError = (
    message: string | undefined,
    field: 'username' | 'password',
  ) => {
    if (!message) return ''
    const key = message === 'required' ? 'fieldRequired.' + field : message
    const translated = t(key as never)
    return translated === key ? message : translated
  }

  const onSubmit = async (values: LoginFormValues) => {
    if (processingRef.current) return
    processingRef.current = true
    setFormAlert('')
    try {
      const data = await authApi.login(values.username, values.password)
      setAuth(data.token, data.user, data.refresh_token)
      // K.3 intended-URL：登录成功 safeNext 校验后跳转，用完即清。
      router.push(safeNext(next))
    } catch (err) {
      const apiErr = err instanceof ApiError ? err : null
      const code = apiErr?.code

      // 字段级：服务端 fieldErrors 回灌（B1.6 层级3）。
      if (apiErr?.fieldErrors?.length) {
        for (const fe of apiErr.fieldErrors) {
          const field = fe.field === 'password' ? 'password' : 'username'
          setError(field, { message: fe.message }, { shouldFocus: true })
        }
        // 其余无字段的错误仍走表单级
        const fieldless = apiErr.fieldErrors.filter((f) => !f.field)
        if (fieldless.length === 0) {
          processingRef.current = false
          return
        }
      }

      // 表单级：凭据错误/封禁/锁定/服务端/网络（B1.6 层级2）。
      switch (code) {
        case ApiErrorCodes.InvalidCredentials:
          setFormAlert(t('formAlert.invalid_credentials'))
          break
        case ApiErrorCodes.UserDisabled:
          setFormAlert(t('formAlert.user_disabled'))
          break
        case ApiErrorCodes.AccountLocked:
        case ApiErrorCodes.RateLimited:
          // B1.8/B1.10：后端消息已带 N 分钟/N 秒（Accept-Language 本地化）。
          setFormAlert(apiErr?.message ?? String(err))
          break
        case ApiErrorCodes.NetworkError:
        case ApiErrorCodes.Timeout:
          setFormAlert(t('formAlert.network'))
          break
        default:
          setFormAlert(
            code === ApiErrorCodes.Internal
              ? t('formAlert.internal')
              : err instanceof Error
                ? err.message
                : String(err),
          )
      }
    } finally {
      processingRef.current = false
    }
  }

  const onGitHubLogin = async () => {
    if (processingRef.current) return
    processingRef.current = true
    try {
      // B1 场景 B1：发起失败 → toast（页面未离开，适合瞬时反馈）。
      await startOAuth(safeNext(next))
    } catch {
      toastManager.add({ type: 'error', title: t('oauthError') })
    } finally {
      processingRef.current = false
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <h1 className="font-display text-headline font-bold tracking-tight">
            {t('title')}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">{t('subtitle')}</p>
        </div>

        {sessionExpired && (
          <div className="mb-4 rounded-md border border-warning/40 bg-warning/10 px-4 py-3 text-sm text-warning">
            {t('sessionExpired')}
          </div>
        )}

        <Card>
          <CardContent className="space-y-4">
            <FormAlert message={formAlert || oauthAlert} />

            <Form onSubmit={handleSubmit(onSubmit)} aria-label={t('title')}>
              <div className="space-y-4">
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
                        render={<Input autoComplete="username" />}
                        placeholder={t('usernamePlaceholder')}
                      />
                      <Field.Error match={!!error}>
                        {error
                          ? translateFieldError(error.message, 'username')
                          : ''}
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
                      <div className="relative">
                        <Field.Control
                          ref={ref}
                          value={value}
                          onValueChange={onChange}
                          onBlur={onBlur}
                          type={showPassword ? 'text' : 'password'}
                          render={<Input autoComplete="current-password" />}
                          placeholder={t('passwordPlaceholder')}
                          className="pr-12"
                        />
                        <button
                          type="button"
                          aria-label={
                            showPassword ? t('hidePassword') : t('showPassword')
                          }
                          onClick={() => setShowPassword((v) => !v)}
                          className="absolute top-1/2 right-3 flex size-11 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
                        >
                          {showPassword ? (
                            <EyeOffIcon className="size-4" />
                          ) : (
                            <EyeIcon className="size-4" />
                          )}
                        </button>
                      </div>
                      <Field.Error match={!!error}>
                        {error
                          ? translateFieldError(error.message, 'password')
                          : ''}
                      </Field.Error>
                    </Field.Root>
                  )}
                />

                <Button
                  type="submit"
                  className="w-full"
                  disabled={isSubmitting || processingRef.current}
                >
                  {isSubmitting ? t('signingIn') : t('loginButton')}
                </Button>
              </div>
            </Form>

            <div className="flex items-center gap-3" aria-hidden="true">
              <span className="h-px flex-1 bg-border" />
              <span className="text-xs text-muted-foreground">{t('or')}</span>
              <span className="h-px flex-1 bg-border" />
            </div>

            <Button
              type="button"
              variant="outline"
              className={cn('w-full', loadingGitHub && 'opacity-70')}
              onClick={onGitHubLogin}
              disabled={loadingGitHub || isSubmitting}
            >
              {loadingGitHub ? (
                <span className="inline-flex items-center gap-2">
                  {t('processingGitHubLogin')}
                </span>
              ) : (
                <span className="inline-flex items-center gap-2">
                  <GithubIcon className="size-4" />
                  {t('githubLogin')}
                </span>
              )}
            </Button>
          </CardContent>
        </Card>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          <Link
            href="/"
            className="underline-offset-4 hover:underline"
            tabIndex={-1}
          >
            markpost
          </Link>
        </p>
      </div>
    </div>
  )
}
