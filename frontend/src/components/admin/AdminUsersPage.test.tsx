import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import { AdminUsersPage } from './AdminUsersPage'

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

describe('AdminUsersPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminUsersPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: /user management/i })
      ).toBeInTheDocument()
    })
  })

  it('displays user rows with username', async () => {
    renderWithProviders(<AdminUsersPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText('user1')).toBeInTheDocument()
    })
  })

  it('displays user role', async () => {
    renderWithProviders(<AdminUsersPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText('User')).toBeInTheDocument()
    })
  })

  it('shows Add User button', async () => {
    renderWithProviders(<AdminUsersPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /add user/i })
      ).toBeInTheDocument()
    })
  })

  it('shows empty state when no users', async () => {
    const { mockAdminUsers } = await import('@/mocks/handlers')
    const original = [...mockAdminUsers]
    mockAdminUsers.length = 0

    renderWithProviders(<AdminUsersPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText(/no users found/i)).toBeInTheDocument()
    })

    mockAdminUsers.push(...original)
  })

  it('renders table headers', async () => {
    renderWithProviders(<AdminUsersPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText('ID')).toBeInTheDocument()
      expect(screen.getByText('Username')).toBeInTheDocument()
      expect(screen.getByText('Role')).toBeInTheDocument()
    })
  })
})
