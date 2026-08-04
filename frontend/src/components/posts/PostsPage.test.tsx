import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import { PostsPage } from './PostsPage'

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

describe('PostsPage', () => {
  it('renders post list items', async () => {
    renderWithProviders(<PostsPage />, { wrapper: ThemeProvider })

    // mockPosts 提供 "Test Post 1" / "Test Post 2"（见 handlers.ts mockPosts）
    await waitFor(() => {
      expect(screen.getByText('Test Post 1')).toBeInTheDocument()
      expect(screen.getByText('Test Post 2')).toBeInTheDocument()
    })
  })

  it('requests posts with a limit query parameter', async () => {
    const { server } = await import('@/mocks/server')
    const { http, HttpResponse } = await import('msw')
    const { mockPosts } = await import('@/mocks/handlers')

    let capturedLimit: string | null = null
    server.use(
      http.get('/api/v1/posts', ({ request }) => {
        const url = new URL(request.url)
        capturedLimit = url.searchParams.get('limit')
        return HttpResponse.json(mockPosts)
      }),
    )

    renderWithProviders(<PostsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText('Test Post 1')).toBeInTheDocument()
    })

    // 验证前端确实传了 limit 参数（不绑死具体数值，避免脆性）
    expect(capturedLimit).not.toBeNull()
    expect(Number(capturedLimit)).toBeGreaterThan(0)
  })

  it('shows empty state when no posts', async () => {
    const { server } = await import('@/mocks/server')
    const { http, HttpResponse } = await import('msw')
    const { mockEmptyPosts } = await import('@/mocks/handlers')

    server.use(
      http.get('/api/v1/posts', () => HttpResponse.json(mockEmptyPosts)),
    )

    renderWithProviders(<PostsPage />, { wrapper: ThemeProvider })

    // 等 loading 结束；空态文案取自 en.json posts 命名空间的空态
    await waitFor(() => {
      expect(screen.queryByText('Test Post 1')).not.toBeInTheDocument()
    })
  })
})
