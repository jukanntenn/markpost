import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import PostsPage from './PostsPage'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.5 帖子列表：搜索 + 四态 + 分页增强（页码/总条数）。
describe('PostsPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<PostsPage />)
    expect(
      await screen.findByRole('heading', { name: /posts/i }),
    ).toBeInTheDocument()
  })

  it('renders post rows with relative links', async () => {
    renderWithProviders(<PostsPage />)
    const link = await screen.findByRole('link', { name: 'Test Post 1' })
    expect(link).toHaveAttribute('href', '/p-qid-1')
    expect(link).toHaveAttribute('target', '_blank')
  })

  it('shows the total count', async () => {
    renderWithProviders(<PostsPage />)
    expect(await screen.findByText('2 total')).toBeInTheDocument()
  })

  it('shows empty state with hint when no posts', async () => {
    const { server } = await import('@/mocks/server')
    const { http, HttpResponse } = await import('msw')
    server.use(
      http.get('/api/v1/posts', () =>
        HttpResponse.json({
          items: [],
          total: 0,
          page: 1,
          limit: 20,
          total_pages: 0,
        }),
      ),
    )
    renderWithProviders(<PostsPage />)
    expect(await screen.findByText('No posts yet')).toBeInTheDocument()
    expect(
      await screen.findByText('Send posts via your Post Key'),
    ).toBeInTheDocument()
  })
})
