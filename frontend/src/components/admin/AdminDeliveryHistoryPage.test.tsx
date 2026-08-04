import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import { AdminDeliveryHistoryPage } from './AdminDeliveryHistoryPage'

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

describe('AdminDeliveryHistoryPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminDeliveryHistoryPage />, {
      wrapper: ThemeProvider,
    })
    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: /delivery history/i }),
      ).toBeInTheDocument()
    })
  })

  it('shows empty state when no history', async () => {
    renderWithProviders(<AdminDeliveryHistoryPage />, {
      wrapper: ThemeProvider,
    })
    await waitFor(() => {
      expect(screen.getByText(/no delivery history/i)).toBeInTheDocument()
    })
  })
})
