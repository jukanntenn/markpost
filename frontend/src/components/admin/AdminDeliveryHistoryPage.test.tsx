import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import AdminDeliveryHistoryPage from './AdminDeliveryHistoryPage'
import { server } from '@/mocks/server'
import { http, HttpResponse } from 'msw'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.8 Admin 投递历史：筛选（用户/渠道/状态）+ 失败详情展开 + 移动卡片。
describe('AdminDeliveryHistoryPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminDeliveryHistoryPage />)
    expect(
      await screen.findByRole('heading', { name: /delivery history/i }),
    ).toBeInTheDocument()
  })

  it('shows empty state when no history', async () => {
    server.use(
      http.get('/api/v1/admin/delivery/history', () =>
        HttpResponse.json({
          items: [],
          total: 0,
          page: 1,
          limit: 20,
          total_pages: 0,
        }),
      ),
    )
    renderWithProviders(<AdminDeliveryHistoryPage />)
    expect(
      await screen.findByText(/no delivery history found/i),
    ).toBeInTheDocument()
  })

  it('renders history rows with post title and channel', async () => {
    server.use(
      http.get('/api/v1/admin/delivery/history', () =>
        HttpResponse.json({
          items: [
            {
              id: 1,
              status: 'delivered',
              last_error: '',
              created_at: '2026-08-10T00:00:00Z',
              channel_id: 1,
              post_title: 'Admin Post A',
              post_qid: 'p-a',
              channel_name: 'Ch1',
              username: 'bob',
            },
          ],
          total: 1,
          page: 1,
          limit: 20,
          total_pages: 1,
        }),
      ),
    )
    renderWithProviders(<AdminDeliveryHistoryPage />)
    expect((await screen.findAllByText('Admin Post A')).length).toBeGreaterThan(
      0,
    )
    expect(screen.getAllByText('Ch1').length).toBeGreaterThan(0)
    expect(screen.getAllByText('bob').length).toBeGreaterThan(0)
  })
})
