import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import AdminDashboardPage from './page'

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

describe('AdminDashboardPage', () => {
  it('renders the dashboard heading', async () => {
    renderWithProviders(<AdminDashboardPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: /dashboard/i })
      ).toBeInTheDocument()
    })
  })

  it('shows loading state initially', () => {
    renderWithProviders(<AdminDashboardPage />, { wrapper: ThemeProvider })
    const dashes = screen.getAllByText('-')
    expect(dashes.length).toBeGreaterThanOrEqual(1)
  })

  it('renders card content after data loads', async () => {
    renderWithProviders(<AdminDashboardPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      const values = screen.getAllByText(/^\d+$/)
      expect(values.length).toBeGreaterThanOrEqual(4)
    })
  })
})
