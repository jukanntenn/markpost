import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/utils'
import { useAuthStore } from '@/stores/auth'
import { server } from '@/mocks/server'
import { useProfileSync } from './useProfileSync'

beforeEach(() => {
  useAuthStore.setState({
    token: 't',
    refreshToken: 'r',
    user: {
      id: 1,
      email: 'admin@example.com',
      username: 'admin',
      role: 'user',
      vip: false,
    },
    _hasHydrated: true,
    sessionExpired: false,
  })
})

// The persisted login-time snapshot must be replaced by whatever /me reports:
// an admin-side vip grant becomes visible on the next page load without
// re-login.
describe('useProfileSync', () => {
  it('overwrites the stored user with the /me profile', async () => {
    server.use(
      http.get('/api/v1/me', () =>
        HttpResponse.json({
          id: 1,
          email: 'admin@example.com',
          username: 'admin',
          role: 'user',
          is_active: true,
          is_email_verified: true,
          vip: true,
        }),
      ),
    )

    renderWithProviders(<Probe />)

    await waitFor(() => {
      expect(useAuthStore.getState().user?.vip).toBe(true)
    })
  })

  it('does not fetch while unauthenticated', async () => {
    useAuthStore.setState({ token: null, user: null })
    let fetched = false
    server.use(
      http.get('/api/v1/me', () => {
        fetched = true
        return HttpResponse.json({})
      }),
    )

    renderWithProviders(<Probe />)

    // Give the disabled query a window in which it must stay silent.
    await new Promise((r) => setTimeout(r, 50))
    expect(fetched).toBe(false)
  })
})

function Probe() {
  useProfileSync()
  return null
}
