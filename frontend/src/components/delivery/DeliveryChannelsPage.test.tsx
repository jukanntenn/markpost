import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import DeliveryChannelsPage from './DeliveryChannelsPage'
import { server } from '@/mocks/server'
import { http, HttpResponse } from 'msw'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.4 渠道列表：空态（可操作 CTA）+ 正常行 + 移动卡片。
describe('DeliveryChannelsPage', () => {
  it('renders the page heading with add action', async () => {
    renderWithProviders(<DeliveryChannelsPage />)
    expect(
      await screen.findByRole('heading', { name: /delivery channels/i }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /add channel/i }),
    ).toBeInTheDocument()
  })

  it('shows actionable empty state when no channels', async () => {
    server.use(
      http.get('/api/v1/delivery/channels', () =>
        HttpResponse.json({ items: [] }),
      ),
    )
    renderWithProviders(<DeliveryChannelsPage />)
    expect(
      await screen.findByText('No delivery channels yet'),
    ).toBeInTheDocument()
    expect(
      screen.getAllByRole('button', { name: /add channel/i }).length,
    ).toBeGreaterThan(0)
  })

  it('renders a channel row with its name', async () => {
    server.use(
      http.get('/api/v1/delivery/channels', () =>
        HttpResponse.json({
          items: [
            {
              id: 1,
              kind: 'feishu',
              name: 'Workgroup',
              enabled: true,
              configuration: {
                webhook_url: 'https://example.com/hook',
                card_link_url: '',
              },
              keywords: 'mark',
              created_at: '2026-08-01T00:00:00Z',
              updated_at: '2026-08-01T00:00:00Z',
            },
          ],
        }),
      ),
    )
    renderWithProviders(<DeliveryChannelsPage />)
    expect((await screen.findAllByText('Workgroup')).length).toBeGreaterThan(0)
    expect(screen.getAllByText('mark').length).toBeGreaterThan(0)
  })
})
