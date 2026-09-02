import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { RetentionHint } from './RetentionHint'

beforeEach(() => {
  mockMatchMedia()
})

// MRFC 2026-09-02-user-facing-retention-visibility: the hint states the
// outcome of the caller's effective policy, per data kind.
describe('RetentionHint', () => {
  it('renders the N-days copy for the resolved policy', async () => {
    renderWithProviders(<RetentionHint kind="posts" />)
    expect(
      await screen.findByText(
        'Data is kept for 7 days, then removed automatically',
      ),
    ).toBeInTheDocument()
  })

  it('renders the forever copy for a zero policy', async () => {
    const { server } = await import('@/mocks/server')
    const { http, HttpResponse } = await import('msw')
    server.use(
      http.get('/api/v1/me/retention', () =>
        HttpResponse.json({ posts_days: 0, history_days: 0 }),
      ),
    )
    renderWithProviders(<RetentionHint kind="posts" />)
    expect(
      await screen.findByText('Data is kept permanently'),
    ).toBeInTheDocument()
  })

  it('renders the history window for kind="history"', async () => {
    const { server } = await import('@/mocks/server')
    const { http, HttpResponse } = await import('msw')
    server.use(
      http.get('/api/v1/me/retention', () =>
        HttpResponse.json({ posts_days: 7, history_days: 30 }),
      ),
    )
    renderWithProviders(<RetentionHint kind="history" />)
    expect(
      await screen.findByText(
        'Data is kept for 30 days, then removed automatically',
      ),
    ).toBeInTheDocument()
  })
})
