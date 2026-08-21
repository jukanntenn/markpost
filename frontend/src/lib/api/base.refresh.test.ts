import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { request } from './base'
import { authApi } from './auth'
import { useAuthStore } from '@/stores/auth'

vi.mock('./auth', () => ({
  authApi: {
    refreshToken: vi.fn(),
  },
}))

const fetchMock = vi.fn()
const refreshTokenMock = vi.mocked(authApi.refreshToken)

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  })

const authHeaderOf = (call: number) => {
  const init = fetchMock.mock.calls[call][1] as RequestInit & {
    headers: Record<string, string>
  }
  return init.headers['Authorization']
}

describe('request token refresh (multi-tab safe)', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock)
    localStorage.clear()
    useAuthStore.setState({
      token: null,
      refreshToken: null,
      user: null,
      sessionExpired: false,
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    fetchMock.mockReset()
    refreshTokenMock.mockReset()
  })

  it('refreshes on 401 and retries with the rotated token', async () => {
    useAuthStore.setState({ token: 'access-1', refreshToken: 'refresh-1' })
    refreshTokenMock.mockResolvedValue({
      token: 'access-2',
      refresh_token: 'refresh-2',
      expires_in: 86400,
    })
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse(401, { code: 'unauthorized', message: 'expired' }),
      )
      .mockResolvedValueOnce(jsonResponse(200, { value: 42 }))

    const result = await request<{ value: number }>('/api/v1/things')

    expect(refreshTokenMock).toHaveBeenCalledExactlyOnceWith('refresh-1')
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(authHeaderOf(1)).toBe('Bearer access-2')
    expect(result).toEqual({ value: 42 })
  })

  it('skips the refresh call when another tab rotated while we queued', async () => {
    useAuthStore.setState({ token: 'access-1', refreshToken: 'refresh-1' })
    // Another tab's rotation landed in localStorage; this tab's in-memory
    // copy is stale. Replaying refresh-1 would trip the backend's reuse
    // detection and revoke every session, so the critical section must
    // re-read storage and adopt the fresh pair instead of calling the API.
    localStorage.setItem(
      'markpost_auth',
      JSON.stringify({
        state: { token: 'access-2', refreshToken: 'refresh-2', user: null },
        version: 0,
      }),
    )
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse(401, { code: 'unauthorized', message: 'expired' }),
      )
      .mockResolvedValueOnce(jsonResponse(200, { value: 7 }))

    const result = await request<{ value: number }>('/api/v1/things')

    expect(refreshTokenMock).not.toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(authHeaderOf(1)).toBe('Bearer access-2')
    expect(result).toEqual({ value: 7 })
  })

  it('marks the session expired when refresh fails', async () => {
    useAuthStore.setState({ token: 'access-1', refreshToken: 'refresh-1' })
    refreshTokenMock.mockRejectedValue(new Error('reuse detected'))
    fetchMock.mockResolvedValue(
      jsonResponse(401, { code: 'unauthorized', message: 'expired' }),
    )

    await expect(request('/api/v1/things')).rejects.toThrow('Session expired')

    const state = useAuthStore.getState()
    expect(state.token).toBeNull()
    expect(state.sessionExpired).toBe(true)
  })
})
