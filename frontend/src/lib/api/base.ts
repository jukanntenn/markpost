import { authApi } from './auth'
import { useAuthStore } from '@/stores/auth'
import { getCurrentLocale } from '@/i18n/current'
import { tClient } from '@/i18n/messages'
import type { FieldError, ApiErrorResponse } from '@/types/api'

// B1.9/B1.10/I.2：前端构造的错误码（附录 D.5，非后端返回）。
export const FRONTEND_ERROR_CODES = {
  network: 'network_error',
  timeout: 'timeout',
  parse: 'parse_error',
} as const

export class ApiError extends Error {
  readonly code?: string
  readonly fieldErrors?: FieldError[]
  // B1.10：429 的 Retry-After 秒数（后端或中间件设置）
  readonly retryAfter?: number

  constructor(response: ApiErrorResponse, options?: { retryAfter?: number }) {
    super(response.message || 'Request failed')
    this.name = 'ApiError'
    this.code = response.code
    this.fieldErrors = response.errors
    this.retryAfter = options?.retryAfter
  }
}

interface RequestOptions extends Omit<RequestInit, 'headers'> {
  skipAuthRefresh?: boolean
  params?: Record<string, string | number>
  json?: unknown
  headers?: Record<string, string>
  // I.2 超时分层：默认 30s，OAuth 等调用单独覆盖。
  timeoutMs?: number
}

let refreshPromise: Promise<boolean> | null = null

async function refreshAccessToken(): Promise<boolean> {
  const { refreshToken, setTokens, logout, markSessionExpired } =
    useAuthStore.getState()

  if (!refreshToken) {
    markSessionExpired()
    logout()
    return false
  }

  try {
    const data = await authApi.refreshToken(refreshToken)
    setTokens(data.token, data.refresh_token)
    return true
  } catch {
    // B1.8 场景 D：refresh 失败 → 会话过期标记 + 本地登出，守卫跳
    // /login?reason=session_expired（登录页展示友好提示）。
    markSessionExpired()
    logout()
    return false
  }
}

async function handleTokenRefresh(): Promise<boolean> {
  if (refreshPromise) {
    return refreshPromise
  }

  refreshPromise = refreshAccessToken().finally(() => {
    refreshPromise = null
  })

  return refreshPromise
}

export function paginationParams(
  page?: number,
  limit?: number,
): Record<string, string | number> {
  const params: Record<string, string | number> = {}
  if (page != null) params.page = page
  if (limit != null) params.limit = limit
  return params
}

export function buildUrl(
  base: string,
  path: string,
  params?: Record<string, string | number>,
): string {
  const normalizedBase = base.endsWith('/') ? base.slice(0, -1) : base
  if (!params || Object.keys(params).length === 0) {
    return `${normalizedBase}${path}`
  }
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    searchParams.set(key, String(value))
  }
  return `${normalizedBase}${path}?${searchParams}`
}

// B1.9 统一捕获：把 fetch 网络层失败转成 ApiError（network_error / timeout /
// parse_error），文案走 i18n（network.* 章节，客户端解析消息）。
function networkApiError(kind: keyof typeof FRONTEND_ERROR_CODES): ApiError {
  const code = FRONTEND_ERROR_CODES[kind]
  return new ApiError({ code, message: tClient(`network.${kind}`) })
}

async function throwApiError(response: Response): Promise<never> {
  let body: ApiErrorResponse
  try {
    body = await response.json()
  } catch {
    throw networkApiError('parse')
  }
  // B1.10：429 读取 Retry-After（秒），供文案倒计时。
  let retryAfter: number | undefined
  if (response.status === 429) {
    const raw = response.headers.get('Retry-After')
    if (raw) {
      const parsed = Number.parseInt(raw, 10)
      if (!Number.isNaN(parsed) && parsed > 0) retryAfter = parsed
    }
  }
  throw new ApiError(body, { retryAfter })
}

async function parseResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    await throwApiError(response)
  }
  // 204 No Content (and any other empty body) has no JSON to parse.
  if (
    response.status === 204 ||
    response.headers.get('content-length') === '0'
  ) {
    return undefined as T
  }
  try {
    return (await response.json()) as T
  } catch {
    throw networkApiError('parse')
  }
}

async function attemptRetry<T>(
  response: Response,
  skipRefresh: boolean,
  retry: () => Promise<Response>,
): Promise<T | undefined> {
  if (response.status !== 401 || skipRefresh) return undefined
  const refreshed = await handleTokenRefresh()
  if (!refreshed) throw new Error('Session expired')
  return parseResponse<T>(await retry())
}

export async function request<T>(
  url: string,
  options: RequestOptions = {},
): Promise<T> {
  const { token } = useAuthStore.getState()
  const {
    skipAuthRefresh = false,
    params,
    json,
    headers: optHeaders,
    timeoutMs = 30_000,
    ...fetchOptions
  } = options

  const fullUrl = buildUrl('', url, params)

  const headers: Record<string, string> = {
    'Accept-Language': getCurrentLocale(),
    ...optHeaders,
  }

  if (json !== undefined) {
    fetchOptions.body = JSON.stringify(json)
    headers['Content-Type'] = 'application/json'
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  // I.2 超时分层：AbortController，超时抛 timeout。
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeoutMs)
  const signal = fetchOptions.signal
    ? AbortSignal.any([controller.signal, fetchOptions.signal])
    : controller.signal

  let response: Response
  try {
    response = await fetch(fullUrl, {
      ...fetchOptions,
      signal,
      headers,
    })
  } catch {
    if (controller.signal.aborted) {
      throw networkApiError('timeout')
    }
    // B1.9：fetch reject（断网/DNS/CORS）→ network_error
    throw networkApiError('network')
  } finally {
    clearTimeout(timer)
  }

  const retried = await attemptRetry<T>(response, skipAuthRefresh, () => {
    headers['Authorization'] = `Bearer ${useAuthStore.getState().token}`
    return fetch(fullUrl, { ...fetchOptions, signal, headers })
  })
  if (retried !== undefined) return retried

  return parseResponse<T>(response)
}
