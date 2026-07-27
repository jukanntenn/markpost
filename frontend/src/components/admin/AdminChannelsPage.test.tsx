import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import { AdminChannelsPage } from './AdminChannelsPage'

vi.mock('@/stores/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockMatchMedia()
})

describe('AdminChannelsPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminChannelsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: /channel management/i })
      ).toBeInTheDocument()
    })
  })

  it('displays channel rows with name and kind', async () => {
    renderWithProviders(<AdminChannelsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText('Alert Channel')).toBeInTheDocument()
      expect(screen.getByText('feishu')).toBeInTheDocument()
    })
  })

  it('shows empty state when no channels', async () => {
    const { mockAdminChannels } = await import('@/mocks/handlers')
    const original = [...mockAdminChannels]
    mockAdminChannels.length = 0

    renderWithProviders(<AdminChannelsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText(/no channels found/i)).toBeInTheDocument()
    })

    mockAdminChannels.push(...original)
  })
})
