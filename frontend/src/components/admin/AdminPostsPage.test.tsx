import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import { AdminPostsPage } from './AdminPostsPage'

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

describe('AdminPostsPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: /post management/i })
      ).toBeInTheDocument()
    })
  })

  it('displays post rows with title', async () => {
    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText('First Post')).toBeInTheDocument()
      expect(screen.getByText('Second Post')).toBeInTheDocument()
    })
  })

  it('renders search input', async () => {
    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search title/i)).toBeInTheDocument()
    })
  })

  it('filters posts by search term', async () => {
    const user = userEvent.setup()
    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText('First Post')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search title/i)
    await user.type(searchInput, 'First')

    await waitFor(() => {
      expect(screen.getByText('First Post')).toBeInTheDocument()
      expect(screen.queryByText('Second Post')).not.toBeInTheDocument()
    })
  })

  it('shows empty state when no posts match search', async () => {
    const user = userEvent.setup()
    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText('First Post')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search title/i)
    await user.type(searchInput, 'NonexistentPost')

    await waitFor(() => {
      expect(screen.getByText(/no posts found/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no posts exist', async () => {
    const { mockAdminPosts } = await import('@/mocks/handlers')
    const original = [...mockAdminPosts]
    mockAdminPosts.length = 0

    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText(/no posts found/i)).toBeInTheDocument()
    })

    mockAdminPosts.push(...original)
  })

  it('renders table headers', async () => {
    renderWithProviders(<AdminPostsPage />, { wrapper: ThemeProvider })
    await waitFor(() => {
      expect(screen.getByText('ID')).toBeInTheDocument()
      expect(screen.getByText('Title')).toBeInTheDocument()
      expect(screen.getByText('Username')).toBeInTheDocument()
    })
  })
})
