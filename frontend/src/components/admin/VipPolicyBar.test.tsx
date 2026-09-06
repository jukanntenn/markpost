import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import { useAuthStore } from '@/stores/auth'
import { server } from '@/mocks/server'
import VipPolicyBar from './VipPolicyBar'

const settingsWith = (days: number | null) => ({
  items: [
    {
      key: 'vip',
      value: { enabled: true },
      updated_by: 1,
      updated_at: '2024-01-01T00:00:00Z',
    },
    ...(days === null
      ? []
      : [
          {
            key: 'vip_retention_days',
            value: { days },
            updated_by: 1,
            updated_at: '2024-01-01T00:00:00Z',
          },
        ]),
  ],
})

function renderBar(settingsDays: number | null) {
  const puts: Array<{ key: string; body: unknown }> = []
  server.use(
    http.get('/api/v1/admin/settings', () =>
      HttpResponse.json(settingsWith(settingsDays)),
    ),
    http.put('/api/v1/admin/settings/:key', async ({ params, request }) => {
      puts.push({ key: params.key as string, body: await request.json() })
      return HttpResponse.json(settingsWith(settingsDays))
    }),
    http.post('/api/v1/admin/retention/impact', () =>
      HttpResponse.json({ users_affected: 0 }),
    ),
  )
  renderWithProviders(<VipPolicyBar />)
  return puts
}

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
  useAuthStore.setState({
    token: 't',
    refreshToken: 'r',
    user: {
      id: 1,
      email: 'admin@example.com',
      username: 'admin',
      role: 'admin',
    },
    _hasHydrated: true,
    sessionExpired: false,
  })
})

// The N-days option used to be a dead click with no saved class default: the
// radio states derived purely from the server value and no local selection
// existed, so nothing rendered, nothing mutated.
describe('VipPolicyBar N-days selection', () => {
  it('reveals and focuses the days input when N days is clicked with no saved default', async () => {
    renderBar(null)

    expect(screen.queryByTestId('vip-class-days-input')).not.toBeInTheDocument()

    fireEvent.click(await screen.findByRole('radio', { name: 'N days' }))

    const input = await screen.findByTestId('vip-class-days-input')
    expect(input).toBeInTheDocument()
    expect(input).toHaveFocus()
  })

  it('saves the typed value on blur', async () => {
    const puts = renderBar(null)

    fireEvent.click(await screen.findByRole('radio', { name: 'N days' }))
    const input = await screen.findByTestId('vip-class-days-input')
    fireEvent.change(input, { target: { value: '30' } })
    fireEvent.blur(input)

    await waitFor(() => {
      expect(puts).toContainEqual({
        key: 'vip_retention_days',
        body: { days: 30 },
      })
    })
  })

  it('commits immediately when the input already holds a valid value', async () => {
    const puts = renderBar(null)

    fireEvent.click(await screen.findByRole('radio', { name: 'N days' }))
    const input = await screen.findByTestId('vip-class-days-input')
    fireEvent.change(input, { target: { value: '14' } })
    fireEvent.blur(input)
    await waitFor(() => expect(puts).toHaveLength(1))

    // Re-clicking the radio with valid text in the field re-commits it.
    fireEvent.click(screen.getByRole('radio', { name: 'N days' }))
    await waitFor(() => expect(puts).toHaveLength(2))
    expect(puts[1]).toEqual({ key: 'vip_retention_days', body: { days: 14 } })
  })

  it('drops the unsaved selection when blurred without a valid value', async () => {
    renderBar(null)

    fireEvent.click(await screen.findByRole('radio', { name: 'N days' }))
    const input = await screen.findByTestId('vip-class-days-input')
    fireEvent.blur(input)

    // Back to the persisted state: input unmounts, follow-global reactivates.
    await waitFor(() => {
      expect(
        screen.queryByTestId('vip-class-days-input'),
      ).not.toBeInTheDocument()
    })
    expect(
      screen.getByRole('radio', { name: 'Follow global' }),
    ).toHaveAttribute('aria-checked', 'true')
  })

  it('still clears to follow-global from a saved days value', async () => {
    const puts = renderBar(30)

    const input = await screen.findByTestId('vip-class-days-input')
    expect(input).toHaveValue(30)

    fireEvent.click(screen.getByRole('radio', { name: 'Follow global' }))
    await waitFor(() => {
      expect(puts).toContainEqual({
        key: 'vip_retention_days',
        body: { days: null },
      })
    })
  })
})
