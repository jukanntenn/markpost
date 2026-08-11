import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import AdminChannelsPage from './AdminChannelsPage'
import { server } from '@/mocks/server'
import { http, HttpResponse } from 'msw'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.7 Admin 渠道管理：桌面表格 + 移动卡片；开关/删除确认文案如实告知影响。
describe('AdminChannelsPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminChannelsPage />)
    expect(
      await screen.findByRole('heading', { name: /channels/i }),
    ).toBeInTheDocument()
  })

  it('shows empty state when no channels', async () => {
    server.use(
      http.get('/api/v1/admin/delivery/channels', () =>
        HttpResponse.json({
          items: [],
          total: 0,
          page: 1,
          limit: 20,
          total_pages: 0,
        }),
      ),
    )
    renderWithProviders(<AdminChannelsPage />)
    expect(await screen.findByText(/no channels found/i)).toBeInTheDocument()
  })

  it('renders the channel name and status badge', async () => {
    renderWithProviders(<AdminChannelsPage />)
    expect(
      (await screen.findAllByText('Alert Channel')).length,
    ).toBeGreaterThan(0)
    expect((await screen.findAllByText('Active')).length).toBeGreaterThan(0)
  })
})
